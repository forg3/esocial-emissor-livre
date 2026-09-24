//go:build !pkcs11

package crypto

import (
	"errors"
	"fmt"
)

// Suporte a certificados A3 (token USB / smartcard via PKCS#11).
//
// Esta compilação NÃO inclui o driver PKCS#11 (build padrão, sem cgo). O
// binário continua suportando A1 normalmente e o fluxo de assinatura com A3 já
// está implementado na camada criptográfica (CertificadoA3 aceita qualquer
// crypto.Signer) — o que falta nesta build é apenas o acesso físico ao token.
//
// Para habilitar o driver, compile com a tag de build:
//
//	go build -tags pkcs11 ./cmd/server
//
// (requer o cabeçalho/libraria pkcs11 e um módulo do fabricante, por exemplo
// /usr/lib/opensc-pkcs11.so para tokens compatíveis com OpenSC.)

// ErrPKCS11Indisponivel indica que a build atual não possui o driver PKCS#11.
var ErrPKCS11Indisponivel = errors.New("suporte a PKCS#11 (certificado A3) não incluído nesta compilação; compile com -tags pkcs11")

// ConfiguracaoTokenA3 descreve o token/smartcard a ser aberto.
type ConfiguracaoTokenA3 struct {
	// Modulo é o caminho da biblioteca PKCS#11 do fabricante
	// (ex.: /usr/lib/opensc-pkcs11.so, /usr/lib/libeTPkcs11.so).
	Modulo string
	// Slot é o identificador do slot (vazio usa o primeiro com token presente).
	Slot string
	// PIN de segurança do token.
	PIN string
	// IDChave opcional (hexadecimal) para selecionar a chave privada.
	IDChave string
}

// PKCS11Disponivel informa se a build inclui o driver PKCS#11.
func PKCS11Disponivel() bool { return false }

// AbrirTokenA3 abre o token PKCS#11 e devolve o certificado A3 pronto para assinar.
func AbrirTokenA3(cfg ConfiguracaoTokenA3) (Certificado, error) {
	if cfg.Modulo == "" {
		return nil, fmt.Errorf("%w (informe o caminho do módulo PKCS#11 do fabricante)", ErrPKCS11Indisponivel)
	}
	return nil, ErrPKCS11Indisponivel
}
