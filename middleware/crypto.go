package middleware

import (
	"bytes"
	"fmt"
	"github.com/QuantumShiftX/golib/config"
	"github.com/QuantumShiftX/golib/crypto"
	"io"
	"net/http"
)

import (
	"encoding/json"
	"log"
)

// CryptoMiddleware 加密中间件（优化版）
func CryptoMiddleware(cfg *config.CryptoConfig) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 检查是否启用加密
			if cfg == nil || !cfg.Enable {
				next.ServeHTTP(w, r)
				return
			}

			// 检查路径是否需要加密
			if !cfg.ShouldEncrypt(r.URL.Path) {
				if cfg.Debug {
					log.Printf("[Crypto] Path %s does not need encryption", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}

			if cfg.Debug {
				log.Printf("[Crypto] Processing encryption for path: %s", r.URL.Path)
			}

			// 解密请求
			if err := decryptHTTPRequest(r, cfg); err != nil {
				if cfg.Debug {
					log.Printf("[Crypto] Request decryption failed: %v", err)
				}
				if cfg.FailOnError {
					http.Error(w, fmt.Sprintf("Request decryption failed: %v", err), http.StatusBadRequest)
					return
				}
			}

			// 创建响应拦截器
			recorder := NewResponseRecorder(w)

			// 执行下一个处理器
			next.ServeHTTP(recorder, r)

			// 加密响应
			if err := encryptHTTPResponse(recorder, w, cfg); err != nil {
				if cfg.Debug {
					log.Printf("[Crypto] Response encryption failed: %v", err)
				}
				if cfg.FailOnError {
					http.Error(w, fmt.Sprintf("Response encryption failed: %v", err), http.StatusInternalServerError)
					return
				}
				// 失败时返回原始响应
				writeOriginalResponse(recorder, w)
			}
		})
	}
}

// decryptHTTPRequest 解密HTTP请求（优化版）
func decryptHTTPRequest(r *http.Request, cfg *config.CryptoConfig) error {
	if r.Method == "GET" || r.Method == "DELETE" || r.Method == "HEAD" {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read request body failed: %w", err)
	}
	r.Body.Close()

	if len(body) == 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
		return nil
	}

	if cfg.Debug {
		log.Printf("[Crypto] Original request body length: %d", len(body))
	}

	// 检查是否为加密格式
	if !crypto.IsEncryptedFormat(body) {
		r.Body = io.NopCloser(bytes.NewReader(body))
		if cfg.Debug {
			log.Println("[Crypto] Request is not in encrypted format, keeping original")
		}
		return nil
	}

	// 解密数据
	var decryptedData interface{}
	if err := crypto.DecryptRequest(body, &decryptedData); err != nil {
		return fmt.Errorf("decrypt request data failed: %w", err)
	}

	// 重新序列化为JSON
	decryptedJSON, err := json.Marshal(decryptedData)
	if err != nil {
		return fmt.Errorf("marshal decrypted data failed: %w", err)
	}

	if cfg.Debug {
		log.Printf("[Crypto] Request decrypted successfully, decrypted length: %d", len(decryptedJSON))
	}

	// 替换请求体
	r.Body = io.NopCloser(bytes.NewReader(decryptedJSON))
	r.ContentLength = int64(len(decryptedJSON))

	return nil
}

// encryptHTTPResponse 加密HTTP响应（优化版）
func encryptHTTPResponse(recorder *ResponseRecorder, w http.ResponseWriter, cfg *config.CryptoConfig) error {
	// 复制头部
	for k, v := range recorder.header {
		w.Header()[k] = v
	}

	status := recorder.status
	responseData := recorder.body.Bytes()

	if cfg.Debug {
		log.Printf("[Crypto] Response data length: %d", len(responseData))
	}

	if len(responseData) == 0 {
		w.WriteHeader(status)
		return nil
	}

	// 解析原始响应
	var originalData interface{}
	if err := json.Unmarshal(responseData, &originalData); err != nil {
		return fmt.Errorf("unmarshal response data failed: %w", err)
	}

	// 加密响应
	encryptedData, err := crypto.QuickEncrypt(originalData)
	if err != nil {
		return fmt.Errorf("encrypt response data failed: %w", err)
	}

	if cfg.Debug {
		log.Println("[Crypto] Response encrypted successfully")
	}

	// 序列化加密响应
	encryptedJSON, err := json.Marshal(encryptedData)
	if err != nil {
		return fmt.Errorf("marshal encrypted response failed: %w", err)
	}

	// 写入响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(encryptedJSON)

	return nil
}

// writeOriginalResponse 写入原始响应（回退）
func writeOriginalResponse(recorder *ResponseRecorder, w http.ResponseWriter) {
	for k, v := range recorder.header {
		w.Header()[k] = v
	}

	w.WriteHeader(recorder.status)
	w.Write(recorder.body.Bytes())
}
