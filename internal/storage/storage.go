package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	Ambiente             int       `json:"ambiente"` // 1 = Produção, 2 = Produção Restrita
	TipoCertificado      string    `json:"tipo_certificado"` // A1 ou A3
	CertificadoPath      string    `json:"certificado_path"`
	CertificadoValidoAte time.Time `json:"certificado_valido_ate"`
	AtualizadoEm         time.Time `json:"atualizado_em"`
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar pasta do banco: %w", err)
	}

	conn, err := sql.Open("sqlite", caminho)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite: %w", err)
	}

	// Habilita WAL mode e foreign keys
	if _, err := conn.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao configurar sqlite: %w", err)
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
	_, err := d.conn.Exec(schema)
	return err
}

// ObterConfiguracao busca o registro singleton de configuração.
func (d *DB) ObterConfiguracao() (*Configuracao, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var c Configuracao
	var validoAte, atualizadoEm sql.NullString
	err := d.conn.QueryRow(`
		SELECT id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate, atualizado_em
		FROM configuracao WHERE id = 'config'
	`).Scan(&c.ID, &c.RazaoSocial, &c.CNPJ, &c.Ambiente, &c.TipoCertificado, &c.CertificadoPath, &validoAte, &atualizadoEm)

	if err == sql.ErrNoRows {
		// Retorna default
		return &Configuracao{
			ID:              "config",
			Ambiente:        2, // Produção Restrita padrão
			TipoCertificado: "A1",
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

	_, err := d.conn.Exec(`
		INSERT INTO configuracao (id, razao_social, cnpj, ambiente, tipo_certificado, certificado_path, certificado_valido_ate, atualizado_em)
		VALUES ('config', ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			razao_social = excluded.razao_social,
			cnpj = excluded.cnpj,
			ambiente = excluded.ambiente,
			tipo_certificado = excluded.tipo_certificado,
			certificado_path = excluded.certificado_path,
			certificado_valido_ate = excluded.certificado_valido_ate,
			atualizado_em = excluded.atualizado_em
	`, c.RazaoSocial, c.CNPJ, c.Ambiente, c.TipoCertificado, c.CertificadoPath, validoStr, agora)
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
	`, e.ID, e.Tipo, e.ColaboradorID, e.Ambiente, e.Status, e.XMLGerado, e.XMLAssinado, e.Protocolo, e.Recibo, e.MensagemRetorno, agora, agora)
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
	TotalColaboradores int
	EventosProntos     int
	EventosAssinados   int
	EventosTransmitidos int
	EventosAceitos     int
	EventosRejeitados  int
	TotalEventos       int
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
