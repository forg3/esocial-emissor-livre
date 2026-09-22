package crypto

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Constantes de algoritmos XMLDSig oficiais do eSocial
const (
	CanonicalizationMethodC14N = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	DigestMethodSHA256         = "http://www.w3.org/2001/04/xmlenc#sha256"
	SignatureMethodRSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	TransformEnveloped         = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	TransformC14N              = "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
	NamespaceXMLDSig           = "http://www.w3.org/2000/09/xmldsig#"
)

// Tipo de nó do DOM
type domNodeType int

const (
	domNodeElement domNodeType = iota
	domNodeText
)

// domAttr representa um atributo XML normal ou declaração de namespace.
type domAttr struct {
	Name      string // Nome completo como aparece na tag (ex: "Id", "xmlns", "xmlns:ds")
	Local     string // Nome local
	Prefix    string // Prefixo (ex: "ds" para "xmlns:ds", ou vazio)
	Namespace string // URI do namespace ao qual o atributo pertence
	Value     string // Valor do atributo
	IsNS      bool   // True se for declaração de namespace (xmlns ou xmlns:prefix)
}

// domNode representa um nó da árvore XML com suporte completo a C14N Inclusive.
type domNode struct {
	Type      domNodeType
	Name      string            // QName do elemento (ex: "evtExpRisco", "eSocial")
	Attrs     []domAttr         // Atributos regulares
	NSAttrs   []domAttr         // Declarações de namespace explícitas neste elemento
	Text      string            // Conteúdo textual se Type == domNodeText
	Children  []*domNode        // Filhos ordenados na sequência do documento
	Parent    *domNode          // Nó pai
	InScopeNS map[string]string // Mapeamento de prefixo -> URI de namespaces em escopo
}

// parseXMLToDOM constrói a árvore de nós do documento XML a partir de um leitor.
func parseXMLToDOM(xmlBytes []byte) (*domNode, error) {
	// Normaliza fins de linha para \n conforme W3C C14N
	normXML := bytes.ReplaceAll(xmlBytes, []byte("\r\n"), []byte("\n"))
	normXML = bytes.ReplaceAll(normXML, []byte("\r"), []byte("\n"))

	decoder := xml.NewDecoder(bytes.NewReader(normXML))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var root *domNode
	var current *domNode

	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("erro de sintaxe no XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			node := &domNode{
				Type:      domNodeElement,
				Name:      montarQName(t.Name),
				InScopeNS: make(map[string]string),
				Parent:    current,
			}

			// Herda namespaces em escopo do pai
			if current != nil {
				for p, uri := range current.InScopeNS {
					node.InScopeNS[p] = uri
				}
			}

			// Processa atributos do elemento
			for _, a := range t.Attr {
				qname := montarQName(a.Name)
				if qname == "xmlns" {
					node.InScopeNS[""] = a.Value
					node.NSAttrs = append(node.NSAttrs, domAttr{
						Name:  "xmlns",
						Local: "",
						Value: a.Value,
						IsNS:  true,
					})
				} else if strings.HasPrefix(qname, "xmlns:") {
					prefix := strings.TrimPrefix(qname, "xmlns:")
					node.InScopeNS[prefix] = a.Value
					node.NSAttrs = append(node.NSAttrs, domAttr{
						Name:   qname,
						Local:  prefix,
						Prefix: "xmlns",
						Value:  a.Value,
						IsNS:   true,
					})
				} else {
					node.Attrs = append(node.Attrs, domAttr{
						Name:      qname,
						Local:     a.Name.Local,
						Prefix:    a.Name.Space,
						Namespace: a.Name.Space,
						Value:     a.Value,
						IsNS:      false,
					})
				}
			}

			if current == nil {
				root = node
			} else {
				current.Children = append(current.Children, node)
			}
			current = node

		case xml.EndElement:
			if current != nil {
				current = current.Parent
			}

		case xml.CharData:
			if current != nil {
				texto := string(t)
				// Preserva conteúdo textual (inclusive whitespace se houver)
				if texto != "" {
					textNode := &domNode{
						Type:   domNodeText,
						Text:   texto,
						Parent: current,
					}
					current.Children = append(current.Children, textNode)
				}
			}

		case xml.Comment, xml.ProcInst, xml.Directive:
			// Ignorados na canonicalização C14N sem comentários
			continue
		}
	}

	if root == nil {
		return nil, errors.New("documento XML vazio ou sem nó raiz")
	}

	return root, nil
}

func montarQName(n xml.Name) string {
	if n.Space != "" && n.Space != "xmlns" {
		// Se houver prefixo definido
		return n.Space + ":" + n.Local
	}
	return n.Local
}

// LocalizarElementoPorID localiza recursivamente o elemento que possui o atributo Id especificado.
func (n *domNode) LocalizarElementoPorID(id string) *domNode {
	idProcurado := strings.TrimPrefix(id, "#")
	if n.Type == domNodeElement {
		for _, a := range n.Attrs {
			if (a.Local == "Id" || a.Local == "id" || a.Name == "Id") && a.Value == idProcurado {
				return n
			}
		}
	}
	for _, child := range n.Children {
		if found := child.LocalizarElementoPorID(id); found != nil {
			return found
		}
	}
	return nil
}

// LocalizarPrimeiroEvento busca o elemento de evento filho imediato da tag raiz eSocial que contenha Id.
func (n *domNode) LocalizarPrimeiroEvento() *domNode {
	if n.Type != domNodeElement {
		return nil
	}

	// Se este nó já possuir um atributo Id começando com "ID", ele é o evento
	for _, a := range n.Attrs {
		if a.Local == "Id" && strings.HasPrefix(a.Value, "ID") {
			return n
		}
	}

	// Busca recursivamente nos filhos
	for _, child := range n.Children {
		if child.Type == domNodeElement {
			for _, a := range child.Attrs {
				if a.Local == "Id" && strings.HasPrefix(a.Value, "ID") {
					return child
				}
			}
		}
	}

	// Se não achou com "ID", pega qualquer elemento que tenha atributo "Id"
	for _, child := range n.Children {
		if child.Type == domNodeElement {
			for _, a := range child.Attrs {
				if a.Local == "Id" && a.Value != "" {
					return child
				}
			}
		}
	}

	return nil
}

// ObterValorID retorna o valor do atributo Id do elemento, se existir.
func (n *domNode) ObterValorID() string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attrs {
		if a.Local == "Id" || a.Local == "id" {
			return a.Value
		}
	}
	return ""
}

// CanonicalizarC14N executa a canonicalização Inclusive C14N (W3C REC-xml-c14n-20010315)
// sobre o elemento do nó, emitindo os namespaces em escopo se ele for a raiz do subconjunto.
func CanonicalizarC14N(node *domNode, isRootOfSubset bool) ([]byte, error) {
	if node == nil {
		return nil, errors.New("nó nulo para canonicalização")
	}

	var buf bytes.Buffer
	renderizarC14N(node, isRootOfSubset, &buf)
	return buf.Bytes(), nil
}

// CanonicalizarXML realiza o parsing do XML e canonicaliza o elemento com o Id informado,
// ou a raiz inteira se id for vazio.
func CanonicalizarXML(xmlBytes []byte, id string) ([]byte, error) {
	root, err := parseXMLToDOM(xmlBytes)
	if err != nil {
		return nil, err
	}

	targetNode := root
	isSubset := false
	if id != "" {
		targetNode = root.LocalizarElementoPorID(id)
		if targetNode == nil {
			return nil, fmt.Errorf("elemento com Id '%s' não encontrado no XML", id)
		}
		isSubset = (targetNode != root)
	}

	return CanonicalizarC14N(targetNode, isSubset)
}

func renderizarC14N(node *domNode, isRootOfSubset bool, buf *bytes.Buffer) {
	if node.Type == domNodeText {
		buf.WriteString(escaparTextoC14N(node.Text))
		return
	}

	buf.WriteByte('<')
	buf.WriteString(node.Name)

	// Coleta declarações de namespaces para este elemento
	namespacesParaRenderizar := make(map[string]string)

	if isRootOfSubset {
		// Na raiz do subconjunto (Inclusive C14N), renderiza todos os namespaces em escopo
		for prefix, uri := range node.InScopeNS {
			if uri != "" {
				namespacesParaRenderizar[prefix] = uri
			}
		}
	} else {
		// Em nós internos, renderiza apenas os declarados explicitamente neste elemento
		for _, ns := range node.NSAttrs {
			namespacesParaRenderizar[ns.Local] = ns.Value
		}
	}

	// Ordena namespaces: "xmlns" (default) primeiro, depois "xmlns:prefix" por ordem alfabética do prefixo
	var chavesNS []string
	temDefaultNS := false
	var defaultNSVal string

	for prefix, uri := range namespacesParaRenderizar {
		if prefix == "" {
			temDefaultNS = true
			defaultNSVal = uri
		} else {
			chavesNS = append(chavesNS, prefix)
		}
	}
	sort.Strings(chavesNS)

	if temDefaultNS && defaultNSVal != "" {
		buf.WriteString(` xmlns="`)
		buf.WriteString(escaparAtributoC14N(defaultNSVal))
		buf.WriteByte('"')
	}

	for _, prefix := range chavesNS {
		buf.WriteString(" xmlns:")
		buf.WriteString(prefix)
		buf.WriteString(`="`)
		buf.WriteString(escaparAtributoC14N(namespacesParaRenderizar[prefix]))
		buf.WriteByte('"')
	}

	// Ordena atributos regulares:
	// 1. Atributos sem namespace URI vêm primeiro, ordenados lexicograficamente pelo nome local.
	// 2. Atributos com namespace URI vêm em seguida, ordenados por URI e depois por nome local.
	attrs := make([]domAttr, len(node.Attrs))
	copy(attrs, node.Attrs)

	sort.Slice(attrs, func(i, j int) bool {
		ai, aj := attrs[i], attrs[j]
		if ai.Namespace == "" && aj.Namespace != "" {
			return true
		}
		if ai.Namespace != "" && aj.Namespace == "" {
			return false
		}
		if ai.Namespace != aj.Namespace {
			return ai.Namespace < aj.Namespace
		}
		return ai.Local < aj.Local
	})

	for _, a := range attrs {
		buf.WriteByte(' ')
		buf.WriteString(a.Name)
		buf.WriteString(`="`)
		buf.WriteString(escaparAtributoC14N(a.Value))
		buf.WriteByte('"')
	}

	buf.WriteByte('>')

	// Renderiza filhos recursivamente
	for _, child := range node.Children {
		renderizarC14N(child, false, buf)
	}

	// Tag de fechamento explícita obrigatória no C14N (mesmo se vazio: <tag></tag>)
	buf.WriteString("</")
	buf.WriteString(node.Name)
	buf.WriteByte('>')
}

func escaparAtributoC14N(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '"':
			b.WriteString("&quot;")
		case '\t':
			b.WriteString("&#x9;")
		case '\n':
			b.WriteString("&#xA;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func escaparTextoC14N(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
