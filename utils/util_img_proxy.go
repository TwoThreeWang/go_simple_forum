package utils

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

func GetImg(c *gin.Context) {
	imgUrl := c.Query("url")
	u, err := url.Parse(imgUrl)
	if err != nil {
		log.Println("Error parsing URL:", err)
		c.Data(http.StatusNotFound, "image/jpeg", nil)
		return
	}

	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0",
		"Host":       u.Host,
		"Referer":    u.Scheme + "://" + u.Host,
	}

	req, err := http.NewRequest("GET", imgUrl, nil)
	if err != nil {
		log.Println("error fetching image:", err)
		c.Data(http.StatusNotFound, "image/jpeg", nil)
		return
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("error fetching image:", err)
		c.Data(http.StatusNotFound, "image/jpeg", nil)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("failed to fetch image. Status code: %d\n", res.StatusCode)
		c.Data(http.StatusNotFound, "image/jpeg", nil)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("error reading image content: ", err)
		c.Data(http.StatusNotFound, "image/jpeg", nil)
		return
	}
	c.Data(http.StatusOK, res.Header.Get("Content-Type"), body)
	return
}
