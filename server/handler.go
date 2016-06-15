package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/jingweno/jqplay/jq"
)

const (
	JSONPayloadLimit   = JSONPayloadLimitMB * OneMB
	JSONPayloadLimitMB = 10
	OneMB              = 1024000
)

type JQHandlerContext struct {
	*Config
	JQ string
}

func (c *JQHandlerContext) Asset(path string) string {
	return fmt.Sprintf("%s/%s", c.Config.AssetHost, path)
}

func (c *JQHandlerContext) ShouldInitJQ() bool {
	return c.JQ != ""
}

type JQHandler struct {
	c *Config
}

func (h *JQHandler) handleIndex(c *gin.Context) {
	c.HTML(200, "index.tmpl", &JQHandlerContext{Config: h.c})
}

func (h *JQHandler) handleJqPost(c *gin.Context) {
	l, _ := c.Get("logger")
	logger := l.(*logrus.Entry)

	if c.Request.ContentLength > JSONPayloadLimit {
		size := float64(c.Request.ContentLength) / OneMB
		err := fmt.Errorf("JSON payload size is %.1fMB, larger than limit %dMB.", size, JSONPayloadLimitMB)
		logger.WithError(err).WithField("size", size).Infof(err.Error())
		c.String(http.StatusExpectationFailed, err.Error())
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, JSONPayloadLimit)

	var jq *jq.JQ
	err := c.BindJSON(&jq)
	if err != nil {
		err = fmt.Errorf("error parsing JSON: %s", err)
		logger.WithError(err).Infof("error parsing JSON: %s", err)
		c.String(422, err.Error())
		return
	}

	// Evaling into ResponseWriter sets the status code to 200
	// appending error message in the end if there's any
	err = jq.Eval(c.Writer)
	if err != nil {
		logger.WithError(err).Infof("error evaluating jq query: %s", err)
		fmt.Fprint(c.Writer, err.Error())
	}
}

func (h *JQHandler) handleJqGet(c *gin.Context) {
	jq := &jq.JQ{
		J: c.Query("j"),
		Q: c.Query("q"),
	}

	var jqData string
	if err := jq.Validate(); err == nil {
		d, err := json.Marshal(jq)
		if err == nil {
			jqData = string(d)
		}
	}

	c.HTML(200, "index.tmpl", &JQHandlerContext{Config: h.c, JQ: jqData})
}
