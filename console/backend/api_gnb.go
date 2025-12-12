package backend

import (
	"fmt"
	"io"
	"net/http"

	"encoding/json"

	"github.com/free-ran-ue/free-ran-ue/console/model"
	"github.com/free-ran-ue/free-ran-ue/constant"
	"github.com/free-ran-ue/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (cs *console) handleConsoleGnbInfo(c *gin.Context) {
	cs.GnbLog.Infoln("Attempting to register gNB")

	if err := authticate(c, cs.jwt.secret); err != nil {
		cs.AuthLog.Warnln(err)
		c.JSON(http.StatusUnauthorized, model.ConsoleGnbInfoResponse{
			Message: err.Error(),
		})
		return
	}

	var request model.ConsoleGnbInfoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		cs.GnbLog.Warnf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, model.ConsoleGnbInfoResponse{
			Message: "Failed to bind JSON",
		})
		return
	}

	uri := fmt.Sprintf("http://%s:%d%s", request.Ip, request.Port, constant.API_REQUEST_GNB_INFO)

	response, err := util.SendHttpRequest(uri, constant.API_REQUEST_GNB_INFO_METHOD, nil, nil)
	if err != nil {
		cs.GnbLog.Warnln(err)
		c.JSON(http.StatusInternalServerError, model.ConsoleGnbInfoResponse{
			Message: err.Error(),
		})
		return
	}

	for key, values := range response.Headers {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Data(response.StatusCode, constant.APPLICATION_JSON, response.Body)
}

func (cs *console) handleConsoleGnbUeNrdcModify(c *gin.Context) {
	cs.GnbLog.Infoln("Attempting to register gNB")

	if err := authticate(c, cs.jwt.secret); err != nil {
		cs.AuthLog.Warnln(err)
		c.JSON(http.StatusUnauthorized, model.ConsoleGnbUeNrdcModifyResponse{
			Message: err.Error(),
		})
		return
	}

	var request model.ConsoleGnbUeNrdcModifyRequest
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		cs.GnbLog.Warnf("Failed to read body: %v", err)
		c.JSON(http.StatusBadRequest, model.ConsoleGnbUeNrdcModifyResponse{
			Message: fmt.Sprintf("Failed to read body: %v", err),
		})
		return
	}

	if err := json.Unmarshal(rawBody, &request); err != nil {
		cs.GnbLog.Warnf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, model.ConsoleGnbUeNrdcModifyResponse{
			Message: fmt.Sprintf("Failed to bind JSON: %v", err),
		})
		return
	}

	uri := fmt.Sprintf("http://%s:%d%s", request.Ip, request.Port, constant.API_REQUEST_GNB_UE_NRDC)

	response, err := util.SendHttpRequest(uri, constant.API_REQUEST_GNB_UE_NRDC_METHOD, nil, rawBody)
	if err != nil {
		cs.GnbLog.Warnln(err)
		c.JSON(http.StatusInternalServerError, model.ConsoleGnbUeNrdcModifyResponse{
			Message: err.Error(),
		})
		return
	}

	for key, values := range response.Headers {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Data(response.StatusCode, constant.APPLICATION_JSON, response.Body)
}
