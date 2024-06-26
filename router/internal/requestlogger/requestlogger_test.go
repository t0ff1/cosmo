package requestlogger

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/cosmo/router/internal/test"
	"github.com/wundergraph/cosmo/router/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRequestLogger(t *testing.T) {

	var buffer bytes.Buffer

	encoder := logging.ZapJsonEncoder()
	writer := bufio.NewWriter(&buffer)

	logger := zap.New(
		zapcore.NewCore(encoder, zapcore.AddSync(writer), zapcore.DebugLevel))

	handler := New(logger)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recovery := handler(handlerFunc)
	rec := httptest.NewRecorder()
	recovery.ServeHTTP(rec, test.NewRequest(http.MethodGet, "/subdir/asdf"))

	writer.Flush()

	assert.Equal(t, http.StatusOK, rec.Code)

	var data map[string]interface{}
	err := json.Unmarshal(buffer.Bytes(), &data)
	assert.Nil(t, err)

	assert.Equal(t, "GET", data["method"])
	assert.Equal(t, float64(200), data["status"])
	assert.Equal(t, "/subdir/asdf", data["msg"])
	assert.Equal(t, "/subdir/asdf", data["path"])

}

func TestRequestFileLogger(t *testing.T) {

	var buffer bytes.Buffer

	fileName := fmt.Sprintf("/tmp/test-logging-%s.log", randStr(32))
	core, err := logging.ZapFileCore(fileName, zapcore.DebugLevel)
	assert.Nil(t, err)
	writer := bufio.NewWriter(&buffer)

	logger := zap.New(core)

	handler := New(logger)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recovery := handler(handlerFunc)
	rec := httptest.NewRecorder()
	recovery.ServeHTTP(rec, test.NewRequest(http.MethodGet, "/subdir/asdf"))

	writer.Flush()

	assert.Equal(t, http.StatusOK, rec.Code)

	logFile, err := os.Open(fileName)
	assert.Nil(t, err)

	reader := bufio.NewReader(logFile)

	line, _, err := reader.ReadLine()
	assert.Nil(t, err)
	var data map[string]interface{}
	json.Unmarshal(line, &data)
	assert.Nil(t, err)
	assert.Equal(t, "GET", data["method"])
	assert.Equal(t, float64(200), data["status"])
	assert.Equal(t, "/subdir/asdf", data["msg"])
	assert.Equal(t, "/subdir/asdf", data["path"])

	err = os.Remove(fileName)
	assert.Nil(t, err)
}

func randStr(length int) string {
	b := make([]byte, length)
	// Read b number of numbers
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}
