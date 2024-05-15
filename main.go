package main

import (
	"bytes"
	"image/png"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kbinani/screenshot"
)

func captureScreen() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0) // Получаем границы первого дисплея
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}

	// Кодирование в PNG
	var imgBuf bytes.Buffer
	err = png.Encode(&imgBuf, img)
	if err != nil {
		return nil, err
	}

	return imgBuf.Bytes(), nil
}

func main() {
	r := gin.Default()

	// Маршрут для отображения экрана в браузере
	r.GET("/", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "image/png")
		imgData, err := captureScreen()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, "image/png", imgData)
	})

	// Маршрут для обновления изображения каждую секунду
	r.GET("/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
		flusher, _ := c.Writer.(http.Flusher)

		for {
			imgData, err := captureScreen()
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return
			}

			// Отправка изображения в поток
			c.Writer.Write([]byte("Content-Type: image/png\n\n"))
			c.Writer.Write(imgData)
			c.Writer.Write([]byte("\n--frame\n"))
			flusher.Flush()

			// Подождать секунду перед захватом следующего кадра
			time.Sleep(1 * time.Second)
		}
	})

	// Запуск сервера
	r.Run(":8081")
}
