package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DB envelopa a conexão SQLite com trava de concorrência se necessário.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Configuracao representa as preferências e certificado do usuário local.
type Configuracao struct {
	ID                   string    `json:"id"`
	RazaoSocial          string    `json:"razao_social"`
	CNPJ                 string    `json:"cnpj"`
	Ambiente             int       `json:"ambiente"`         // 1 = Produção, 2 = Produção Restrita
	TipoCertificado      string    `json:"tipo_certificado"` // A1 ou A3
	CertificadoPath      string    `json:"certificado_path"`
	CertificadoValidoAte time.Time `json:"certificado_valido_ate"`
	// ModoTransmissao define se o envio ao eSocial é "simulado" (padrão) ou "real"
	// (webservice oficial via mTLS com o certificado A1).
	ModoTransmissao string    `json:"modo_transmissao"`
	AtualizadoEm    time.Time `json:"atualizado_em"`
}

// Colaborador representa um trabalhador registrado localmente.
type Colaborador struct {
	ID           string    `json:"id"`
	Nome         string    `json:"nome"`
	CPF          string    `json:"cpf"`
	Matricula    string    `json:"matricula"`
	Cargo        string    `json:"cargo"`
	CBO          string    `json:"cbo"`
	DataAdmissao string    `json:"data_admissao"`
	Setor        string    `json:"setor"`
	Status       string    `json:"status"` // ativo, inativo
	CriadoEm     time.Time `json:"criado_em"`
}

// Evento representa um evento do eSocial gerado no sistema.
type Evento struct {
	ID              string    `json:"id"`
	Tipo            string    `json:"tipo"` // S-1000, S-2210, S-2220, S-2240
	ColaboradorID   string    `json:"colaborador_id"`
	ColaboradorNome string    `json:"colaborador_nome,omitempty"`
	ColaboradorCPF  string    `json:"colaborador_cpf,omitempty"`
	Ambiente        int       `json:"ambiente"` // 1 = Producao, 2 = Restrita
	Status          string    `json:"status"`   // pronto, assinado, transmitido, aceito, rejeitado
	XMLGerado       string    `json:"xml_gerado"`
	XMLAssinado     string    `json:"xml_assinado"`
	Protocolo       string    `json:"protocolo"`
	Recibo          string    `json:"recibo"`
	MensagemRetorno string    `json:"mensagem_retorno"`
	CriadoEm        time.Time `json:"criado_em"`
	AtualizadoEm    time.Time `json:"atualizado_em"`
}

// Abrir inicializa o banco SQLite local no diretório especificado ou cria um novo.
func Abrir(caminho string) (*DB, error) {
	dir := filepath.Dir(caminho)
	// 0700: o banco contém dados pessoais (CPF) e deve ser acessível apenas ao dono
	// do processo (achado B-01).
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("falha ao criar pasta do banco: %w", err)
	}
	// Corrige permissões de diretórios preexistentes criados com 0755.
	_ = os.Chmod(dir, 0700)

	conn, err := sql.Open("sqlite", caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite: %w", err)
	}

	// Habilita WAL mode e foreign keys
	if _, err := conn.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao configurar sqlite: %w", err)
	}

	// Restringe o arquivo do banco ao dono do processo (feito após a primeira
	// consulta, quando o arquivo já existe de fato).
	if err := os.Chmod(caminho, 0600); err != nil && !os.IsNotExist(err) {
		conn.Close()
		return nil, fmt.Errorf("falha ao ajustar permissoes do banco: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrar(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha nas migrações: %w", err)
	}

	return db, nil
}

// Fechar encerra o banco.
func (d *DB) Fechar() error {
	return d.conn.Close()
}

func (d *DB) migrar() error {
	schema := `
	CREATE TABLE IF NOT EXISTS configuracao (
		id TEXT PRIMARY KEY,
		razao_social TEXT NOT NULL DEFAULT '',
		cnpj TEXT NOT NULL DEFAULT '',
		ambiente INTEGER NOT NULL DEFAULT 2,
		tipo_certificado TEXT NOT NULL DEFAULT 'A1',
		certificado_path TEXT NOT NULL DEFAULT '',
		certificado_valido_ate TEXT,
		atualizado_em TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS colaborador (
		id TEXT PRIMARY KEY,
		nome TEXT NOT NULL,
		cpf TEXT NOT NULL UNIQUE,
		matricula TEXT NOT NULL DEFAULT '',
		cargo TEXT NOT NULL DEFAULT '',
		cbo TEXT NOT NULL DEFAULT '',
		data_admissao TEXT NOT NULL DEFAULT '',
		setor TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ativo',
		criado_em TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS evento (
		id TEXT PRIMARY KEY,
		tipo TEXT NOT NULL,
		colaborador_id TEXT,
		ambiente INTEGER NOT NULL DEFAULT 2,
		status TEXT NOT NULL DEFAULT 'pronto',
		xml_gerado TEXT NOT NULL,
		xml_assinado TEXT NOT NULL DEFAULT '',
		protocolo TEXT NOT NULL DEFAULT '',
		recibo TEXT NOT NULL DEFAULT '',
		mensagem_retorno TEXT NOT NULL DEFAULT '',
		criado_em TEXT NOT NULL,
		atualizado_em TEXT NOT NULL,
		FOREIGN KEY (colaborador_id) REFERENCES colaborador(id) ON DELETE SET NULL
	);
	`
	if _, err := d.conn.Exec(schema); err != nil {
		return err
	}
	if err := migrarRetornos(d); err != nil {
		return err
	}
	if err := migrarPerfis(d); err != nil {
		return err
	}
	return migrarModoTransmissao(d)
}

// migrarModoTransmissao garante a coluna de modo de transmissão na configuração
// ("simulado" por padrão; "real" habilita o envio pelo webservice oficial).
func migrarModoTransmissao(d *DB) error {
	_, err := d.conn.Exec(`ALTER TABLE configuracao ADD COLUMN modo_transmissao TEXT NOT NULL DEFAULT 'simulado'`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

// ObterConfiguracao busca o registro singleton de configuração.
func (d *DB) ObterConfiguracao() (*Configuracao, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var c Configuracao
	var validoAte, atualizadoEm sql.NullString
	err := d.conn.QueryRow(`
		SELECT id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate, atualizado_em,
		       COALESCE(modo_transmissao, 'simulado')
		FROM configuracao WHERE id = 'config'
	`).Scan(&c.ID, &c.RazaoSocial, &c.CNPJ, &c.Ambiente, &c.TipoCertificado, &c.CertificadoPath, &validoAte, &atualizadoEm, &c.ModoTransmissao)

	if err == sql.ErrNoRows {
		// Retorna default
		return &Configuracao{
			ID:              "config",
			Ambiente:        2, // Produção Restrita padrão
			TipoCertificado: "A1",
			ModoTransmissao: "simulado",
			AtualizadoEm:    time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	if validoAte.Valid {
		c.CertificadoValidoAte, _ = time.Parse(time.RFC3339, validoAte.String)
	}
	if atualizadoEm.Valid {
		c.AtualizadoEm, _ = time.Parse(time.RFC3339, atualizadoEm.String)
	}
	return &c, nil
}

// SalvarConfiguracao atualiza os dados da empresa e certificado.
func (d *DB) SalvarConfiguracao(c *Configuracao) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	agora := time.Now().Format(time.RFC3339)
	validoStr := ""
	if !c.CertificadoValidoAte.IsZero() {
		validoStr = c.CertificadoValidoAte.Format(time.RFC3339)
	}

	modo := c.ModoTransmissao
	if modo != "real" {
		modo = "simulado"
	}
	_, err := d.conn.Exec(`
		INSERT INTO configuracao (id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate, atualizado_em, modo_transmissao)
		VALUES ('config', ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			razao_social = excluded.razao_social,
			cnpj = excluded.cnpj,
			ambiente = excluded.ambiente,
			tipo_certificado = excluded.tipo_certificado,
			certificado_path = excluded.certificado_path,
			certificado_valido_ate = excluded.certificado_valido_ate,
			atualizado_em = excluded.atualizado_em,
			modo_transmissao = excluded.modo_transmissao
	`, c.RazaoSocial, c.CNPJ, c.Ambiente, c.TipoCertificado, c.CertificadoPath, validoStr, agora, modo)
	return err
}

// ListarColaboradores retorna todos os colaboradores ativos.
func (d *DB) ListarColaboradores() ([]Colaborador, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT id, nome, cpf, matricula, cargo, cbo, data_admissao, setor, status, criado_em
		FROM colaborador ORDER BY nome ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Colaborador
	for rows.Next() {
		var c Colaborador
		var criadoEm string
		if err := rows.Scan(&c.ID, &c.Nome, &c.CPF, &c.Matricula, &c.Cargo, &c.CBO, &c.DataAdmissao, &c.Setor, &c.Status, &criadoEm); err != nil {
			return nil, err
		}
		c.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
		lista = append(lista, c)
	}
	return lista, nil
}

// ObterColaborador busca um colaborador por ID.
func (d *DB) ObterColaborador(id string) (*Colaborador, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var c Colaborador
	var criadoEm string
	err := d.conn.QueryRow(`
		SELECT id, nome, cpf, matricula, cargo, cbo, data_admissao, setor, status, criado_em
		FROM colaborador WHERE id = ?
	`, id).Scan(&c.ID, &c.Nome, &c.CPF, &c.Matricula, &c.Cargo, &c.CBO, &c.DataAdmissao, &c.Setor, &c.Status, &criadoEm)
	if err != nil {
		return nil, err
	}
	c.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
	return &c, nil
}

// SalvarColaborador insere ou atualiza um colaborador.
func (d *DB) SalvarColaborador(c *Colaborador) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	agora := time.Now().Format(time.RFC3339)
	_, err := d.conn.Exec(`
		INSERT INTO colaborador (id, nome, cpf, matricula, cargo, cbo, data_admissao, setor, status, criado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			nome = excluded.nome,
			cpf = excluded.cpf,
			matricula = excluded.matricula,
			cargo = excluded.cargo,
			cbo = excluded.cbo,
			data_admissao = excluded.data_admissao,
			setor = excluded.setor,
			status = excluded.status
	`, c.ID, c.Nome, c.CPF, c.Matricula, c.Cargo, c.CBO, c.DataAdmissao, c.Setor, c.Status, agora)
	return err
}

// SalvarEvento insere ou atualiza um evento eSocial.
func (d *DB) SalvarEvento(e *Evento) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	agora := time.Now().Format(time.RFC3339)
	var colabID any
	if e.ColaboradorID != "" {
		colabID = e.ColaboradorID
	}
	_, err := d.conn.Exec(`
		INSERT INTO evento (id, tipo, colaborador_id, ambiente, status, xml_gerado, xml_assinado, protocolo, recibo, mensagem_retorno, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			xml_gerado = excluded.xml_gerado,
			xml_assinado = excluded.xml_assinado,
			protocolo = excluded.protocolo,
			recibo = excluded.recibo,
			mensagem_retorno = excluded.mensagem_retorno,
			atualizado_em = excluded.atualizado_em
	`, e.ID, e.Tipo, colabID, e.Ambiente, e.Status, e.XMLGerado, e.XMLAssinado, e.Protocolo, e.Recibo, e.MensagemRetorno, agora, agora)
	return err
}

// ListarEventos retorna os eventos ordenados pelos mais recentes.
func (d *DB) ListarEventos() ([]Evento, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT e.id, e.tipo, e.colaborador_id, COALESCE(c.nome, ''), COALESCE(c.cpf, ''),
		       e.ambiente, e.status, e.xml_gerado, e.xml_assinado, e.protocolo, e.recibo, e.mensagem_retorno,
		       e.criado_em, e.atualizado_em
		FROM evento e
		LEFT JOIN colaborador c ON c.id = e.colaborador_id
		ORDER BY e.criado_em DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Evento
	for rows.Next() {
		var e Evento
		var colabID sql.NullString
		var criadoEm, atualizadoEm string
		if err := rows.Scan(
			&e.ID, &e.Tipo, &colabID, &e.ColaboradorNome, &e.ColaboradorCPF,
			&e.Ambiente, &e.Status, &e.XMLGerado, &e.XMLAssinado, &e.Protocolo, &e.Recibo, &e.MensagemRetorno,
			&criadoEm, &atualizadoEm,
		); err != nil {
			return nil, err
		}
		if colabID.Valid {
			e.ColaboradorID = colabID.String
		}
		e.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
		e.AtualizadoEm, _ = time.Parse(time.RFC3339, atualizadoEm)
		lista = append(lista, e)
	}
	return lista, nil
}

// ObterEvento busca um evento por ID.
func (d *DB) ObterEvento(id string) (*Evento, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var e Evento
	var colabID sql.NullString
	var criadoEm, atualizadoEm string
	err := d.conn.QueryRow(`
		SELECT e.id, e.tipo, e.colaborador_id, COALESCE(c.nome, ''), COALESCE(c.cpf, ''),
		       e.ambiente, e.status, e.xml_gerado, e.xml_assinado, e.protocolo, e.recibo, e.mensagem_retorno,
		       e.criado_em, e.atualizado_em
		FROM evento e
		LEFT JOIN colaborador c ON c.id = e.colaborador_id
		WHERE e.id = ?
	`, id).Scan(
		&e.ID, &e.Tipo, &colabID, &e.ColaboradorNome, &e.ColaboradorCPF,
		&e.Ambiente, &e.Status, &e.XMLGerado, &e.XMLAssinado, &e.Protocolo, &e.Recibo, &e.MensagemRetorno,
		&criadoEm, &atualizadoEm,
	)
	if err != nil {
		return nil, err
	}
	if colabID.Valid {
		e.ColaboradorID = colabID.String
	}
	e.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
	e.AtualizadoEm, _ = time.Parse(time.RFC3339, atualizadoEm)
	return &e, nil
}

// ExcluirColaborador remove um colaborador pelo ID.
func (d *DB) ExcluirColaborador(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec(`DELETE FROM colaborador WHERE id = ?`, id)
	return err
}

// ExcluirEvento remove um evento pelo ID.
func (d *DB) ExcluirEvento(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec(`DELETE FROM evento WHERE id = ?`, id)
	return err
}

// AtualizarStatusEvento atualiza o status, recibo, protocolo e mensagem de retorno de um evento.
func (d *DB) AtualizarStatusEvento(id, status, recibo, protocolo, msg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	agora := time.Now().Format(time.RFC3339)
	_, err := d.conn.Exec(`
		UPDATE evento SET
			status = ?,
			recibo = CASE WHEN ? != '' THEN ? ELSE recibo END,
			protocolo = CASE WHEN ? != '' THEN ? ELSE protocolo END,
			mensagem_retorno = ?,
			atualizado_em = ?
		WHERE id = ?
	`, status, recibo, recibo, protocolo, protocolo, msg, agora, id)
	return err
}

// ResumoKPI consolida contadores para o painel de controle.
type ResumoKPI struct {
	TotalColaboradores  int
	EventosProntos      int
	EventosAssinados    int
	EventosSimulados    int
	EventosTransmitidos int
	EventosAceitos      int
	EventosRejeitados   int
	TotalEventos        int
}

// ObterResumoKPI computa os contadores para exibição no dashboard.
func (d *DB) ObterResumoKPI() (*ResumoKPI, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var kpi ResumoKPI
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM colaborador WHERE status = 'ativo'`).Scan(&kpi.TotalColaboradores); err != nil {
		return nil, err
	}

	rows, err := d.conn.Query(`SELECT status, COUNT(*) FROM evento GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var st string
		var cnt int
		if err := rows.Scan(&st, &cnt); err != nil {
			return nil, err
		}
		kpi.TotalEventos += cnt
		switch st {
		case "pronto":
			kpi.EventosProntos += cnt
		case "assinado":
			kpi.EventosAssinados += cnt
		case "simulado":
			kpi.EventosSimulados += cnt
		case "transmitido":
			kpi.EventosTransmitidos += cnt
		case "aceito":
			kpi.EventosAceitos += cnt
		case "rejeitado":
			kpi.EventosRejeitados += cnt
		}
	}
	return &kpi, nil
}

// -----------------------------------------------------------------------------
// RETORNO DE TOTALIZAÇÃO (S-5001/S-5002/S-5003 e S-5011/S-5012/S-5013)
// -----------------------------------------------------------------------------

// RetornoTotalizacao é um evento de totalização recebido do eSocial, persistido
// para conferência de débitos previdenciários e FGTS.
type RetornoTotalizacao struct {
	ID            string            `json:"id"`
	Tipo          string            `json:"tipo"`
	DescricaoTipo string            `json:"descricao_tipo"`
	PerApur       string            `json:"per_apur"`
	CPFTrab       string            `json:"cpf_trab"`
	Matricula     string            `json:"matricula"`
	NRRecibo      string            `json:"nr_recibo"`
	Valores       map[string]string `json:"valores"`
	TotalCentavos int64             `json:"total_centavos"`
	ImportadoEm   time.Time         `json:"importado_em"`
}

func migrarRetornos(d *DB) error {
	_, err := d.conn.Exec(`
	CREATE TABLE IF NOT EXISTS retorno_totalizacao (
		id TEXT PRIMARY KEY,
		tipo TEXT NOT NULL,
		descricao_tipo TEXT NOT NULL DEFAULT '',
		per_apur TEXT NOT NULL DEFAULT '',
		cpf_trab TEXT NOT NULL DEFAULT '',
		matricula TEXT NOT NULL DEFAULT '',
		nr_recibo TEXT NOT NULL DEFAULT '',
		valores TEXT NOT NULL DEFAULT '{}',
		total_centavos INTEGER NOT NULL DEFAULT 0,
		importado_em TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_retorno_per_apur ON retorno_totalizacao (per_apur);
	CREATE INDEX IF NOT EXISTS idx_retorno_cpf ON retorno_totalizacao (cpf_trab);
	`)
	return err
}

// SalvarRetornoTotalizacao insere ou atualiza um totalizador importado.
func (d *DB) SalvarRetornoTotalizacao(r *RetornoTotalizacao) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	valores, err := json.Marshal(r.Valores)
	if err != nil {
		return fmt.Errorf("falha ao serializar valores do retorno: %w", err)
	}
	_, err = d.conn.Exec(`
		INSERT INTO retorno_totalizacao (id, tipo, descricao_tipo, per_apur, cpf_trab, matricula, nr_recibo, valores, total_centavos, importado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			per_apur = excluded.per_apur,
			cpf_trab = excluded.cpf_trab,
			matricula = excluded.matricula,
			nr_recibo = excluded.nr_recibo,
			valores = excluded.valores,
			total_centavos = excluded.total_centavos,
			importado_em = excluded.importado_em
	`, r.ID, r.Tipo, r.DescricaoTipo, r.PerApur, r.CPFTrab, r.Matricula, r.NRRecibo, string(valores), r.TotalCentavos, r.ImportadoEm.Format(time.RFC3339))
	return err
}

// ListarRetornosTotalizacao devolve os totalizadores importados, opcionalmente
// filtrados por período de apuração.
func (d *DB) ListarRetornosTotalizacao(perApur string) ([]RetornoTotalizacao, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	consulta := `SELECT id, tipo, descricao_tipo, per_apur, cpf_trab, matricula, nr_recibo, valores, total_centavos, importado_em
	             FROM retorno_totalizacao`
	args := []any{}
	if perApur != "" {
		consulta += ` WHERE per_apur = ?`
		args = append(args, perApur)
	}
	consulta += ` ORDER BY per_apur DESC, tipo ASC, cpf_trab ASC`

	rows, err := d.conn.Query(consulta, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []RetornoTotalizacao
	for rows.Next() {
		var r RetornoTotalizacao
		var valores, importadoEm string
		if err := rows.Scan(&r.ID, &r.Tipo, &r.DescricaoTipo, &r.PerApur, &r.CPFTrab, &r.Matricula,
			&r.NRRecibo, &valores, &r.TotalCentavos, &importadoEm); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(valores), &r.Valores)
		r.ImportadoEm, _ = time.Parse(time.RFC3339, importadoEm)
		lista = append(lista, r)
	}
	return lista, nil
}

// ExcluirRetornoTotalizacao remove um totalizador importado.
func (d *DB) ExcluirRetornoTotalizacao(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec(`DELETE FROM retorno_totalizacao WHERE id = ?`, id)
	return err
}

// ResumoRetornoTotalizacao consolida os valores importados por período.
type ResumoRetornoTotalizacao struct {
	PerApur                string
	Registros              int
	Trabalhadores          int
	TotalCentavos          int64
	PrevidenciarioCentavos int64
	FGTSCentavos           int64
	IRRFCentavos           int64
}

// ObterResumoRetornos consolida os totalizadores por período de apuração.
func (d *DB) ObterResumoRetornos() ([]ResumoRetornoTotalizacao, error) {
	lista, err := d.ListarRetornosTotalizacao("")
	if err != nil {
		return nil, err
	}

	indice := map[string]*ResumoRetornoTotalizacao{}
	cpfs := map[string]map[string]bool{}
	var ordem []string

	for _, r := range lista {
		res, ok := indice[r.PerApur]
		if !ok {
			res = &ResumoRetornoTotalizacao{PerApur: r.PerApur}
			indice[r.PerApur] = res
			cpfs[r.PerApur] = map[string]bool{}
			ordem = append(ordem, r.PerApur)
		}
		res.Registros++
		res.TotalCentavos += r.TotalCentavos
		if r.CPFTrab != "" {
			cpfs[r.PerApur][r.CPFTrab] = true
		}
		switch r.Tipo {
		case "evtBasesTrab", "evtCS", "evtInfoContrib":
			res.PrevidenciarioCentavos += r.TotalCentavos
		case "evtFGTS", "evtFGTSCons", "evtBasesFGTS":
			res.FGTSCentavos += r.TotalCentavos
		case "evtIrrfBenef", "evtIrrf":
			res.IRRFCentavos += r.TotalCentavos
		}
	}

	var resumo []ResumoRetornoTotalizacao
	for _, per := range ordem {
		r := indice[per]
		r.Trabalhadores = len(cpfs[per])
		resumo = append(resumo, *r)
	}
	return resumo, nil
}

// -----------------------------------------------------------------------------
// PERFIS MULTI-EMPRESA (múltiplos certificados e procurações eletrônicas)
// -----------------------------------------------------------------------------

// PerfilEmpresa representa uma empresa/contribuinte com o seu próprio
// certificado digital e, opcionalmente, os dados do procurador eletrônico.
type PerfilEmpresa struct {
	ID                   string    `json:"id"`
	RazaoSocial          string    `json:"razao_social"`
	CNPJ                 string    `json:"cnpj"`
	Ambiente             int       `json:"ambiente"`
	TipoCertificado      string    `json:"tipo_certificado"`
	CertificadoPath      string    `json:"certificado_path"`
	CertificadoValidoAte time.Time `json:"certificado_valido_ate"`
	ModoTransmissao      string    `json:"modo_transmissao"`
	ProcuradorNome       string    `json:"procurador_nome"`
	ProcuradorDoc        string    `json:"procurador_doc"`
	Ativo                bool      `json:"ativo"`
	CriadoEm             time.Time `json:"criado_em"`
	AtualizadoEm         time.Time `json:"atualizado_em"`
}

func migrarPerfis(d *DB) error {
	_, err := d.conn.Exec(`
	CREATE TABLE IF NOT EXISTS perfil_empresa (
		id TEXT PRIMARY KEY,
		razao_social TEXT NOT NULL DEFAULT '',
		cnpj TEXT NOT NULL DEFAULT '',
		ambiente INTEGER NOT NULL DEFAULT 2,
		tipo_certificado TEXT NOT NULL DEFAULT 'A1',
		certificado_path TEXT NOT NULL DEFAULT '',
		certificado_valido_ate TEXT NOT NULL DEFAULT '',
		procurador_nome TEXT NOT NULL DEFAULT '',
		procurador_doc TEXT NOT NULL DEFAULT '',
		ativo INTEGER NOT NULL DEFAULT 0,
		criado_em TEXT NOT NULL,
		atualizado_em TEXT NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_perfil_cnpj ON perfil_empresa (cnpj);
	`)
	return err
}

// ListarPerfis devolve todos os perfis cadastrados (ativo primeiro).
func (d *DB) ListarPerfis() ([]PerfilEmpresa, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.conn.Query(`
		SELECT id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate,
		       procurador_nome, procurador_doc, ativo, criado_em, atualizado_em
		FROM perfil_empresa
		ORDER BY ativo DESC, razao_social ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []PerfilEmpresa
	for rows.Next() {
		var p PerfilEmpresa
		var validoAte, criadoEm, atualizadoEm string
		if err := rows.Scan(&p.ID, &p.RazaoSocial, &p.CNPJ, &p.Ambiente, &p.TipoCertificado, &p.CertificadoPath,
			&validoAte, &p.ProcuradorNome, &p.ProcuradorDoc, &p.Ativo, &criadoEm, &atualizadoEm); err != nil {
			return nil, err
		}
		if validoAte != "" {
			p.CertificadoValidoAte, _ = time.Parse(time.RFC3339, validoAte)
		}
		p.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
		p.AtualizadoEm, _ = time.Parse(time.RFC3339, atualizadoEm)
		lista = append(lista, p)
	}
	return lista, nil
}

// ObterPerfil busca um perfil pelo identificador.
func (d *DB) ObterPerfil(id string) (*PerfilEmpresa, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var p PerfilEmpresa
	var validoAte, criadoEm, atualizadoEm string
	err := d.conn.QueryRow(`
		SELECT id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate,
		       procurador_nome, procurador_doc, ativo, criado_em, atualizado_em
		FROM perfil_empresa WHERE id = ?
	`, id).Scan(&p.ID, &p.RazaoSocial, &p.CNPJ, &p.Ambiente, &p.TipoCertificado, &p.CertificadoPath,
		&validoAte, &p.ProcuradorNome, &p.ProcuradorDoc, &p.Ativo, &criadoEm, &atualizadoEm)
	if err != nil {
		return nil, err
	}
	if validoAte != "" {
		p.CertificadoValidoAte, _ = time.Parse(time.RFC3339, validoAte)
	}
	p.CriadoEm, _ = time.Parse(time.RFC3339, criadoEm)
	p.AtualizadoEm, _ = time.Parse(time.RFC3339, atualizadoEm)
	return &p, nil
}

// ObterPerfilAtivo devolve o perfil marcado como ativo (nil quando não houver).
func (d *DB) ObterPerfilAtivo() (*PerfilEmpresa, error) {
	d.mu.RLock()
	var id string
	err := d.conn.QueryRow(`SELECT id FROM perfil_empresa WHERE ativo = 1 LIMIT 1`).Scan(&id)
	d.mu.RUnlock()
	if err != nil {
		return nil, nil
	}
	return d.ObterPerfil(id)
}

// SalvarPerfil insere ou atualiza um perfil de empresa.
func (d *DB) SalvarPerfil(p *PerfilEmpresa) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	agora := time.Now().Format(time.RFC3339)
	validoAte := ""
	if !p.CertificadoValidoAte.IsZero() {
		validoAte = p.CertificadoValidoAte.Format(time.RFC3339)
	}
	ativo := 0
	if p.Ativo {
		ativo = 1
	}

	_, err := d.conn.Exec(`
		INSERT INTO perfil_empresa (id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path,
			certificado_valido_ate, procurador_nome, procurador_doc, ativo, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			razao_social = excluded.razao_social,
			cnpj = excluded.cnpj,
			ambiente = excluded.ambiente,
			tipo_certificado = excluded.tipo_certificado,
			certificado_path = excluded.certificado_path,
			certificado_valido_ate = excluded.certificado_valido_ate,
			procurador_nome = excluded.procurador_nome,
			procurador_doc = excluded.procurador_doc,
			ativo = excluded.ativo,
			atualizado_em = excluded.atualizado_em
	`, p.ID, p.RazaoSocial, p.CNPJ, p.Ambiente, p.TipoCertificado, p.CertificadoPath, validoAte,
		p.ProcuradorNome, p.ProcuradorDoc, ativo, agora, agora)
	return err
}

// ExcluirPerfil remove um perfil de empresa.
func (d *DB) ExcluirPerfil(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec(`DELETE FROM perfil_empresa WHERE id = ?`, id)
	return err
}

// AtivarPerfil marca um único perfil como ativo e sincroniza a configuração
// global usada pelos geradores de eventos (CNPJ, razão social, ambiente e
// certificado), garantindo compatibilidade com o fluxo existente.
func (d *DB) AtivarPerfil(id string) error {
	perfil, err := d.ObterPerfil(id)
	if err != nil {
		return fmt.Errorf("perfil não encontrado: %w", err)
	}

	d.mu.Lock()
	if _, err := d.conn.Exec(`UPDATE perfil_empresa SET ativo = 0, atualizado_em = ?`, time.Now().Format(time.RFC3339)); err != nil {
		d.mu.Unlock()
		return err
	}
	if _, err := d.conn.Exec(`UPDATE perfil_empresa SET ativo = 1, atualizado_em = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339), id); err != nil {
		d.mu.Unlock()
		return err
	}
	d.mu.Unlock()

	cfg, err := d.ObterConfiguracao()
	if err != nil {
		return err
	}
	cfg.RazaoSocial = perfil.RazaoSocial
	cfg.CNPJ = perfil.CNPJ
	cfg.Ambiente = perfil.Ambiente
	cfg.TipoCertificado = perfil.TipoCertificado
	cfg.CertificadoPath = perfil.CertificadoPath
	cfg.CertificadoValidoAte = perfil.CertificadoValidoAte
	cfg.AtualizadoEm = time.Now()
	return d.SalvarConfiguracao(cfg)
}
