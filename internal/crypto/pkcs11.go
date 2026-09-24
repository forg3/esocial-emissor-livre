//go:build pkcs11

package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/miekg/pkcs11"
)

// Suporte a certificados A3 (token USB / smartcard via PKCS#11).
//
// Esta compilação INCLUI o driver PKCS#11 (habilitada com `-tags pkcs11`).
// A chave privada nunca sai do token: a assinatura é delegada ao hardware pelo
// crypto.Signer retornado por AbrirTokenA3.

// PKCS11Disponivel informa se a build inclui o driver PKCS#11.
func PKCS11Disponivel() bool { return true }

// ConfiguracaoTokenA3 descreve o token/smartcard a ser aberto.
type ConfiguracaoTokenA3 struct {
	Modulo  string
	Slot    string
	PIN     string
	IDChave string
}

// AbrirTokenA3 abre o módulo PKCS#11, localiza o certificado e a chave privada
// do token e devolve um Certificado A3 cujo Signer assina dentro do hardware.
func AbrirTokenA3(cfg ConfiguracaoTokenA3) (Certificado, error) {
	if strings.TrimSpace(cfg.Modulo) == "" {
		return nil, fmt.Errorf("informe o caminho do módulo PKCS#11 do fabricante")
	}
	if cfg.PIN == "" {
		return nil, fmt.Errorf("informe o PIN de segurança do token")
	}

	ctx := pkcs11.New(cfg.Modulo)
	if ctx == nil {
		return nil, fmt.Errorf("não foi possível carregar o módulo PKCS#11 %q", cfg.Modulo)
	}
	if err := ctx.Initialize(); err != nil {
		return nil, fmt.Errorf("falha ao inicializar o PKCS#11 (%s): %w", cfg.Modulo, err)
	}

	slots, err := ctx.GetSlotList(true)
	if err != nil || len(slots) == 0 {
		ctx.Finalize()
		ctx.Destroy()
		return nil, fmt.Errorf("nenhum token/smartcard presente nos slots do módulo %q", cfg.Modulo)
	}
	slot := slots[0]
	if cfg.Slot != "" {
		for _, s := range slots {
			if fmt.Sprintf("%d", s) == strings.TrimSpace(cfg.Slot) {
				slot = s
				break
			}
		}
	}

	ses, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION)
	if err != nil {
		ctx.Finalize()
		ctx.Destroy()
		return nil, fmt.Errorf("falha ao abrir sessão no token: %w", err)
	}
	if err := ctx.Login(ses, pkcs11.CKU_USER, cfg.PIN); err != nil {
		ctx.CloseSession(ses)
		ctx.Finalize()
		ctx.Destroy()
		return nil, fmt.Errorf("PIN incorreto ou token bloqueado: %w", err)
	}

	// Localiza o certificado X.509 (CKO_CERTIFICATE)
	err = ctx.FindObjectsInit(ses, []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_CERTIFICATE),
	})
	if err != nil {
		return nil, encerrarSessao(ctx, ses, fmt.Errorf("falha ao procurar certificados no token: %w", err))
	}
	encontrados, _, err := ctx.FindObjects(ses, 10)
	_ = ctx.FindObjectsFinal(ses)
	if err != nil || len(encontrados) == 0 {
		return nil, encerrarSessao(ctx, ses, fmt.Errorf("nenhum certificado encontrado no token"))
	}

	var folha *x509.Certificate
	for _, obj := range encontrados {
		atributos, err := ctx.GetAttributeValue(ses, obj, []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_VALUE, 0),
		})
		if err != nil || len(atributos) == 0 {
			continue
		}
		cert, err := x509.ParseCertificate(atributos[0].Value)
		if err != nil {
			continue
		}
		if folha == nil || cert.NotAfter.After(folha.NotAfter) {
			folha = cert
		}
	}
	if folha == nil {
		return nil, encerrarSessao(ctx, ses, fmt.Errorf("certificado do token não pôde ser interpretado (X.509)"))
	}

	// Localiza a chave privada (CKO_PRIVATE_KEY)
	filtrosChave := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
	}
	if cfg.IDChave != "" {
		id, err := hex.DecodeString(strings.TrimSpace(cfg.IDChave))
		if err != nil {
			return nil, encerrarSessao(ctx, ses, fmt.Errorf("identificador de chave inválido (use hexadecimal): %w", err))
		}
		filtrosChave = append(filtrosChave, pkcs11.NewAttribute(pkcs11.CKA_ID, id))
	}
	if err := ctx.FindObjectsInit(ses, filtrosChave); err != nil {
		return nil, encerrarSessao(ctx, ses, fmt.Errorf("falha ao procurar a chave privada: %w", err))
	}
	chaves, _, err := ctx.FindObjects(ses, 2)
	_ = ctx.FindObjectsFinal(ses)
	if err != nil || len(chaves) == 0 {
		return nil, encerrarSessao(ctx, ses, fmt.Errorf("nenhuma chave privada encontrada no token (verifique o PIN e o identificador da chave)"))
	}

	signer := &signerToken{ctx: ctx, sess: ses, chave: chaves[0], chavePub: folha.PublicKey}
	return NovoCertificadoA3(signer, folha, nil), nil
}

func encerrarSessao(ctx *pkcs11.Ctx, sess pkcs11.SessionHandle, err error) error {
	_ = ctx.Logout(sess)
	_ = ctx.CloseSession(sess)
	ctx.Finalize()
	ctx.Destroy()
	return err
}

// signerToken implementa crypto.Signer delegando a assinatura ao hardware.
type signerToken struct {
	ctx      *pkcs11.Ctx
	sess     pkcs11.SessionHandle
	chave    pkcs11.ObjectHandle
	chavePub crypto.PublicKey
}

// Public devolve a chave pública do certificado do token.
func (s *signerToken) Public() crypto.PublicKey { return s.chavePub }

// Sign assina o digest com a chave privada residente no token (RSA PKCS#1 v1.5).
func (s *signerToken) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if _, ok := s.chavePub.(*rsa.PublicKey); !ok {
		return nil, fmt.Errorf("somente chaves RSA são suportadas para certificados A3 nesta versão")
	}

	mecanismo := pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil)
	if err := s.ctx.SignInit(s.sess, []*pkcs11.Mechanism{mecanismo}, s.chave); err != nil {
		return nil, fmt.Errorf("falha ao iniciar a assinatura no token: %w", err)
	}

	// Para RSA o PKCS#11 assina o DigestInfo; o pacote miekg/pkcs11 espera os
	// bytes do hash já prefixados quando o mecanismo é CKM_RSA_PKCS.
	assinatura, err := s.ctx.Sign(s.sess, digest)
	if err != nil {
		return nil, fmt.Errorf("falha ao assinar no token: %w", err)
	}
	return assinatura, nil
}
