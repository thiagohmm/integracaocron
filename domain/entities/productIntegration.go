package entities

import "time"

// ProductInJson represents the main product integration JSON structure
type ProductInJson struct {
	ProdutosSelect []ProductSelectIntegration `json:"produtosSelect"`
	Pesavel        string                     `json:"pesavel"`
}

// ProductSelectIntegration represents a product selection with all its details for integration
type ProductSelectIntegration struct {
	Desc      string         `json:"desc"`
	DescEcf   string         `json:"descEcf"`
	PitStop   string         `json:"pitstop"`
	Subclasse string         `json:"subclasse"`
	Nivel1    string         `json:"nivel1"`
	Depto     string         `json:"depto"`
	CodRMS    string         `json:"codrms"`
	Status    string         `json:"status"`
	DescMarca string         `json:"descMarca"`
	Ind       string         `json:"ind"`
	CodBarras []CodigoBarras `json:"codBarras"`
	Embalagem []Embalagem    `json:"embalagem"`
	Pesavel   string         `json:"pesavel"`
}

// CodigoBarras represents a barcode structure
type CodigoBarras struct {
	CBarra string `json:"cBarra"`
	Princ  string `json:"princ"`
	Tipo   string `json:"tipo"`
}

// Embalagem represents packaging information
type Embalagem struct {
	EAN  string `json:"ean"`
	Qtde string `json:"qtde"`
}

// ProductNew represents a new product structure for internal processing
type ProductNew struct {
	IdProduto                *int               `json:"id_produto"`
	DescricaoProduto         string             `json:"descricao_produto"`
	DescricaoCupom           string             `json:"descricao_cupom"`
	PitStop                  int                `json:"pitstop"`
	IdEstruturaMercadologica *int               `json:"id_estrutura_mercadologica"`
	IdNivel1EstrMerc         *int               `json:"id_nivel1_estr_merc"`
	IdNivel2EstrMerc         *int               `json:"id_nivel2_estr_merc"`
	IdNivel3EstrMerc         *int               `json:"id_nivel3_estr_merc"`
	Notabilidade             string             `json:"notabilidade"`
	CodigoRMS                *int               `json:"codigo_rms"`
	Ativo                    bool               `json:"ativo"`
	MarkUp                   *float64           `json:"markup"`
	PeriodoShelfLife         string             `json:"periodo_shelf_life"`
	ShelfLife                *int               `json:"shelf_life"`
	TipoProduto              *int               `json:"tipo_produto"`
	Producao                 *int               `json:"producao"`
	DataUltimaAtualizacao    *time.Time         `json:"data_ultima_atualizacao"`
	IdMarca                  *int               `json:"id_marca"`
	ConteudoEmbalagem        *int               `json:"conteudo_embalagem"`
	Embalagens               []ProductPackaging `json:"embalagens"`
	ForaMix                  *int               `json:"fora_mix"`
	Regional                 *int               `json:"regional"`
	IdUnidadeMedida          *int               `json:"id_unidade_medida"`
	DiretorioAnexo           string             `json:"diretorio_anexo"`
	Gift                     *int               `json:"gift"`
	Observacao               string             `json:"observacao"`
	ReferenciaFabricante     string             `json:"referencia_fabricante"`
}

// ProductPackaging represents product packaging information
type ProductPackaging struct {
	IdProduto           *int   `json:"id_produto"`
	CodigoBarras        string `json:"codigo_barras"`
	Principal           bool   `json:"principal"`
	QuantidadeEmbalagem int    `json:"quantidade_embalagem"`
	IdUnidadeMedida     *int   `json:"id_unidade_medida"`
	TipoCodigoBarras    string `json:"tipo_codigo_barras"`
}

// Product represents the complete product entity
type Product struct {
	IdProduto                  *int     `json:"id_produto" db:"ID_PRODUTO"`
	Ativo                      int      `json:"ativo" db:"ATIVO"`
	ConteudoEmbalagem          *int     `json:"conteudo_embalagem" db:"CONTEUDO_EMBALAGEM"`
	DescricaoCupom             string   `json:"descricao_cupom" db:"DESCRICAO_CUPOM"`
	DescricaoProduto           string   `json:"descricao_produto" db:"DESCRICAO_PRODUTO"`
	DiretorioAnexo             string   `json:"diretorio_anexo" db:"DIRETORIO_ANEXO"`
	Gift                       *int     `json:"gift" db:"GIFT"`
	IdEstruturaMercadologica   *int     `json:"id_estrutura_mercadologica" db:"ID_ESTRUTURA_MERCADOLOGICA"`
	IdMarca                    *int     `json:"id_marca" db:"ID_MARCA"`
	IdNivel1EstrMerc           *int     `json:"id_nivel1_estr_merc" db:"ID_NIVEL1_ESTR_MERC"`
	IdNivel2EstrMerc           *int     `json:"id_nivel2_estr_merc" db:"ID_NIVEL2_ESTR_MERC"`
	IdNivel3EstrMerc           *int     `json:"id_nivel3_estr_merc" db:"ID_NIVEL3_ESTR_MERC"`
	IdUnidadeMedida            *int     `json:"id_unidade_medida" db:"ID_UNIDADE_MEDIDA"`
	MarkUp                     *float64 `json:"markup" db:"MARKUP"`
	Notabilidade               string   `json:"notabilidade" db:"NOTABILIDADE"`
	Observacao                 string   `json:"observacao" db:"OBSERVACAO"`
	PeriodoShelfLife           string   `json:"periodo_shelf_life" db:"PERIODO_SHELF_LIFE"`
	ReferenciaFabricante       string   `json:"referencia_fabricante" db:"REFERENCIA_FABRICANTE"`
	ShelfLife                  *int     `json:"shelf_life" db:"SHELF_LIFE"`
	TipoProduto                *int     `json:"tipo_produto" db:"TIPO_PRODUTO"`
	Producao                   *int     `json:"producao" db:"PRODUCAO"`
	PitStop                    *int     `json:"pitstop" db:"PITSTOP"`
	ForaMix                    *int     `json:"fora_mix" db:"FORA_MIX"`
	Regional                   *int     `json:"regional" db:"REGIONAL"`
	ProduDataUltimaAtualizacao string   `json:"produ_data_ultima_atualizacao" db:"PRODU_DATA_ULTIMA_ATUALIZACAO"`
	CodigoRMS                  *int     `json:"codigo_rms" db:"CODIGO_RMS"`
	Industria                  string   `json:"industria" db:"INDUSTRIA"`
	IdEstruturaCompra          *int     `json:"id_estrutura_compra" db:"ID_ESTRUTURA_COMPRA"`
}

// MarketingStructure represents marketing structure information
type MarketingStructure struct {
	IdEstruturaMercadologica *int   `json:"id_estrutura_mercadologica" db:"ID_ESTRUTURA_MERCADOLOGICA"`
	IdNivelPai               *int   `json:"id_nivel_pai" db:"ID_NIVEL_PAI"`
	IdDepartamento           *int   `json:"id_departamento" db:"ID_DEPARTAMENTO"`
	IdSecao                  *int   `json:"id_secao" db:"ID_SECAO"`
	DescricaoEstrutura       string `json:"descricao_estrutura" db:"DESCRICAO_ESTRUTURA"`
}

// Brand represents brand information
type Brand struct {
	IdMarca       *int   `json:"id_marca" db:"ID_MARCA"`
	NomeMarca     string `json:"nome_marca" db:"NOME_MARCA"`
	IdIndustria   *int   `json:"id_industria" db:"ID_INDUSTRIA"`
	StatusMarca   int    `json:"status_marca" db:"STATUS_MARCA"`
	NomeIndustria string `json:"nome_industria,omitempty"`
}

// Industry represents industry information
type Industry struct {
	IdIndustria     *int   `json:"id_industria" db:"ID_INDUSTRIA"`
	NomeIndustria   string `json:"nome_industria" db:"NOME_INDUSTRIA"`
	StatusIndustria int    `json:"status_industria" db:"STATUS_INDUSTRIA"`
}

// IntegrRmsProductIn represents the RMS product integration input
type IntegrRmsProductIn struct {
	IprID           *int       `json:"ipr_id" db:"IPR_ID"`
	JSON            string     `json:"json" db:"JSON"`
	DataRecebimento *time.Time `json:"data_recebimento" db:"DATARECEBIMENTO"`
}

// LogIntegrRMS represents the integration log
type LogIntegrRMS struct {
	LirID               *int       `json:"lir_id" db:"LIR_ID"`
	Transacao           string     `json:"transacao" db:"TRANSACAO"`
	Tabela              string     `json:"tabela" db:"TABELA"`
	DataRecebimento     *time.Time `json:"data_recebimento" db:"DATARECEBIMENTO"`
	DataProcessamento   *time.Time `json:"data_processamento" db:"DATAPROCESSAMENTO"`
	StatusProcessamento int        `json:"status_processamento" db:"STATUSPROCESSAMENTO"`
	JSON                string     `json:"json" db:"JSON"`
	DescricaoErro       string     `json:"descricao_erro" db:"DESCRICAOERRO"`
}

// LogValidate represents validation log structure
type LogValidate struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// JsonProductSegment represents product segment JSON structure for integration
type JsonProductSegment struct {
	Cod                string             `json:"cod"`
	Desc               string             `json:"desc"`
	DescEcf            string             `json:"descEcf"`
	DescMarca          string             `json:"descMarca"`
	UnidMed            string             `json:"unidMed"`
	DescUnidMed        string             `json:"descUnidMed"`
	Depto              string             `json:"depto"`
	NmSecao            string             `json:"nmSecao"`
	Nivel1             int                `json:"nivel1"`
	Nivel2             int                `json:"nivel2"`
	Nivel3             int                `json:"nivel3"`
	Nivel4             int                `json:"nivel4"`
	CodBarrasPrincipal string             `json:"codBarrasPrincipal"`
	CodBarras          []JsonCodigoBarras `json:"codBarras"`
	MarkUp             *float64           `json:"markUp"`
	Status             string             `json:"status"`
	Utilizacao         *int               `json:"utilizacao"`
	Producao           int                `json:"producao"`
	DtAlt              time.Time          `json:"dtAlt"`
	StCompra           int                `json:"stCompra"`
}

// JsonCodigoBarras represents barcode JSON structure
type JsonCodigoBarras struct {
	EAN string `json:"ean"`
}

// UnitOfMeasurement represents unit of measurement information
type UnitOfMeasurement struct {
	IdUnidadeMedida        *int   `json:"id_unidade_medida" db:"ID_UNIDADE_MEDIDA"`
	CodigoUnidadeMedida    string `json:"codigo_unidade_medida" db:"CODIGO_UNIDADE_MEDIDA"`
	DescricaoUnidadeMedida string `json:"descricao_unidade_medida" db:"DESCRICAO_UNIDADE_MEDIDA"`
}

// Department represents department information
type Department struct {
	IdDepartamento   *int   `json:"id_departamento" db:"ID_DEPARTAMENTO"`
	NomeDepartamento string `json:"nome_departamento" db:"NOME_DEPARTAMENTO"`
}

// Section represents section information
type Section struct {
	IdSecao   *int   `json:"id_secao" db:"ID_SECAO"`
	NomeSecao string `json:"nome_secao" db:"NOME_SECAO"`
}

// QueueMessage represents a message structure for queue operations
type QueueMessage struct {
	Tabela string        `json:"tabela"`
	Fields []string      `json:"fields"`
	Values []interface{} `json:"values"`
}

// Constants for product integration
const (
	CONST_TRUE    = "true"
	CONST_FALSE   = "false"
	CONST_ATIVO   = 1
	CONST_ATIVO_A = "A"
	CONST_EMPTY   = ""

	CB_BARRA_EAN   = "EAN"
	CB_BARRA_EAN13 = "EAN13"
	CB_INTERNO     = "INTERNO"

	UNIDADE_MEDIDA_KG = 2 // Usually KG unit ID
	UNIDADE_MEDIDA_UN = 1 // Usually UN unit ID

	NOTABILIDADE = "Não Notável"

	MSG_IMPORT_PRODUCT_NOT_FOUND = "Produto não encontrado"
)
