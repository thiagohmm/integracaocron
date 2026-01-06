package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/pkg/logger"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

type HTTPHandler struct {
	listener *Listener
}

func NewHTTPHandler(listener *Listener) *HTTPHandler {
	return &HTTPHandler{
		listener: listener,
	}
}

type IntegrationRequest struct {
	TipoIntegracao string                 `json:"tipoIntegracao"`
	Dados          map[string]interface{} `json:"dados,omitempty"`
}

type IntegrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (h *HTTPHandler) ProcessIntegration(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "http.process_integration")
	defer span.End()

	var req IntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Invalid request format: %v", err)
		tracing.RecordError(ctx, err)
		c.JSON(http.StatusBadRequest, IntegrationResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
		})
		return
	}

	tracing.AddStringAttribute(ctx, "integration.type", req.TipoIntegracao)
	logger.Info(ctx, "REST API - Processando integração: %s", req.TipoIntegracao)

	err, _ := h.processIntegration(ctx, req.TipoIntegracao, req.Dados)
	if err != nil {
		logger.Error(ctx, "REST API - Erro ao processar integração: %v", err)
		tracing.RecordError(ctx, err)
		tracing.SetStatus(ctx, trace.StatusCodeError, err.Error())
		c.JSON(http.StatusInternalServerError, IntegrationResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	logger.Info(ctx, "REST API - Integração processada com sucesso: %s", req.TipoIntegracao)
	c.JSON(http.StatusOK, IntegrationResponse{
		Success: true,
		Message: "Integração processada com sucesso",
	})
}

func (h *HTTPHandler) processIntegration(ctx context.Context, tipoIntegracao string, dados map[string]interface{}) (error, string) {
	ctx, span := tracing.StartSpan(ctx, "integration.process")
	defer span.End()

	tracing.AddStringAttribute(ctx, "integration.type", tipoIntegracao)
	tracing.AddIntAttribute(ctx, "integration.data_count", len(dados))

	switch tipoIntegracao {
	case "promocao", "Promocao":
		return h.processPromotion(ctx, dados)
	case "produto", "Produto":
		return h.processProduct(ctx)
	case "promocao_normalizacao", "PromocaoNormalizacao":
		return h.processPromotionNormalization(ctx)
	case "mover", "productNetworkMain", "product_network_main":
		return h.processProductNetworkMain(ctx)
	default:
		err := fmt.Errorf("tipo de processo desconhecido: %s", tipoIntegracao)
		logger.Error(ctx, "Tipo de processo desconhecido: %s", tipoIntegracao)
		return err, ""
	}
}

func (h *HTTPHandler) processPromotion(ctx context.Context, dados map[string]interface{}) (error, string) {
	ctx, span := tracing.StartSpan(ctx, "integration.promotion")
	defer span.End()

	logger.Info(ctx, "Iniciando processamento de promoção")

	if len(dados) == 0 || (dados["ipm_id"] == nil && dados["IPM_ID"] == nil) {
		logger.Info(ctx, "Nenhum dado específico de promoção fornecido, processando todas as promoções pendentes do banco")
		tracing.AddEvent(ctx, "processing_all_pending_promotions")
		err := h.listener.PromocaoUC.ProcessarTodasPromocoesPendentes()
		if err != nil {
			logger.Error(ctx, "Erro ao processar promoções pendentes: %v", err)
			return fmt.Errorf("erro ao processar promoções pendentes: %w", err), ""
		}
	} else {
		logger.Info(ctx, "Dados específicos de promoção fornecidos, processando promoção individual")
		tracing.AddEvent(ctx, "processing_specific_promotion")
		var promocao entities.Promotion
		promocaoBytes, err := json.Marshal(dados)
		if err != nil {
			logger.Error(ctx, "Erro ao serializar dados de promoção: %v", err)
			return fmt.Errorf("erro ao serializar dados de promoção: %w", err), ""
		}
		if err := json.Unmarshal(promocaoBytes, &promocao); err != nil {
			logger.Error(ctx, "Erro ao desserializar dados para entities.Promotion: %v", err)
			return fmt.Errorf("erro ao desserializar dados para entities.Promotion: %w", err), ""
		}

		if promocao.IPM_ID == 0 {
			logger.Error(ctx, "IPM_ID inválido (0), não é possível processar promoção vazia")
			return fmt.Errorf("IPM_ID inválido: não é possível processar promoção com ID 0"), ""
		}

		tracing.AddInt64Attribute(ctx, "promotion.ipm_id", int64(promocao.IPM_ID))
		err = h.listener.PromocaoUC.ProcessarPromocao(promocao)
		if err != nil {
			logger.Error(ctx, "Erro ao processar promoção: %v", err)
			return fmt.Errorf("erro ao processar promoção: %w", err), ""
		}

		err = h.listener.IntegrationUc.IntegrationJob()
		if err != nil {
			logger.Error(ctx, "Erro ao processar integração: %v", err)
			return fmt.Errorf("erro ao processar integração: %w", err), ""
		}
	}

	logger.Info(ctx, "Processamento de promoção concluído")
	return nil, ""
}

func (h *HTTPHandler) processProduct(ctx context.Context) (error, string) {
	ctx, span := tracing.StartSpan(ctx, "integration.product")
	defer span.End()

	logger.Info(ctx, "Iniciando processamento de produto")

	if h.listener.ProductIntegrationUC == nil {
		logger.Error(ctx, "ProductIntegrationUC não foi inicializado")
		return fmt.Errorf("ProductIntegrationUC não foi inicializado"), ""
	}

	success, err := h.listener.ProductIntegrationUC.ImportProductIntegration()
	if err != nil {
		logger.Error(ctx, "Erro ao processar integração de produtos: %v", err)
		return fmt.Errorf("erro ao processar integração de produtos: %w", err), ""
	}

	if !success {
		logger.Warn(ctx, "Integração de produtos concluída com alguns erros")
		return fmt.Errorf("integração de produtos concluída com alguns erros"), ""
	}

	logger.Info(ctx, "Processamento de produto concluído com sucesso")
	return nil, ""
}

func (h *HTTPHandler) processPromotionNormalization(ctx context.Context) (error, string) {
	ctx, span := tracing.StartSpan(ctx, "integration.promotion_normalization")
	defer span.End()

	logger.Info(ctx, "Iniciando normalização de promoções")

	if h.listener.PromotionNormalizationUC == nil {
		logger.Error(ctx, "PromotionNormalizationUC não foi inicializado")
		return fmt.Errorf("PromotionNormalizationUC não foi inicializado"), ""
	}

	if h.listener.IntegrationUc == nil {
		logger.Error(ctx, "IntegrationUc não foi inicializado")
		return fmt.Errorf("IntegrationUc não foi inicializado"), ""
	}

	result, err := h.listener.PromotionNormalizationUC.NormalizePromotions()
	if err != nil {
		logger.Error(ctx, "Erro ao processar normalização de promoções: %v", err)
		return fmt.Errorf("erro ao processar normalização de promoções: %w", err), ""
	}

	if !result.Success {
		logger.Warn(ctx, "Normalização de promoções concluída com alguns erros: %s", result.Message)
		return fmt.Errorf("normalização de promoções concluída com alguns erros: %s", result.Message), ""
	}

	tracing.AddIntAttribute(ctx, "normalization.processed_count", result.ProcessedCount)
	tracing.AddIntAttribute(ctx, "normalization.updated_count", result.UpdatedCount)
	tracing.AddIntAttribute(ctx, "normalization.removed_duplicates", result.TotalRemovedDuplicates)

	logger.Info(ctx, "Normalização de promoções concluída com sucesso. Processados: %d, Atualizados: %d, Duplicatas removidas: %d",
		result.ProcessedCount, result.UpdatedCount, result.TotalRemovedDuplicates)
	return nil, ""
}

func (h *HTTPHandler) processProductNetworkMain(ctx context.Context) (error, string) {
	ctx, span := tracing.StartSpan(ctx, "integration.product_network_main")
	defer span.End()

	logger.Info(ctx, "Iniciando processo ProductNetworkMain")

	if h.listener.IntegrationUc == nil {
		logger.Error(ctx, "IntegrationUc não foi inicializado")
		return fmt.Errorf("IntegrationUc não foi inicializado"), ""
	}

	dataCorte := time.Now()
	tracing.AddStringAttribute(ctx, "product_network.data_corte", dataCorte.Format(time.RFC3339))
	err := h.listener.productNetworkMain(dataCorte)
	if err != nil {
		logger.Error(ctx, "Erro ao executar ProductNetworkMain: %v", err)
		return fmt.Errorf("erro ao executar ProductNetworkMain: %w", err), ""
	}

	logger.Info(ctx, "Processo ProductNetworkMain concluído com sucesso")
	return nil, ""
}