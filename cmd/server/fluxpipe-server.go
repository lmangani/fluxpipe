package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/metrico/fluxpipe/service"
	"github.com/metrico/fluxpipe/static"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var APPNAME = "fluxpipe"

func postQuery(c echo.Context) error {

	c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
	c.Response().Header().Set("x-fluxpipe-cloud", "qxip")

	content := c.Request().Header.Get("Content-Type")
	s, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	if strings.Contains(string(s), "buckets()") {
		// fake bucket to make grafana happy
		buckets := "#datatype,string,string,string,string,string,string,long\n" +
			"#default,_result,,,,,,\n" +
			",result,table,name,id,organizationID,retentionPolicy,retentionPeriod\n" +
			",_result,0,_fluxpipe,aa9f5aa08895152b,03dbe8db13d17000,,604800000000000\n" +
			"\n"
		return c.String(http.StatusOK, buckets)
	}

	if strings.Contains(content, "json") {
		json_map := make(map[string]interface{})
		err := json.Unmarshal(s, &json_map)
		if err != nil {
			return err
		} else {
			q := json_map["query"]
			query := fmt.Sprintf("%v", q)
			res, err := service.RunE(c.Request().Context(), query)
			if err != nil {
				c.Response().Header().Set(echo.HeaderContentType, "application/json; charset=utf-8")
				c.Response().Header().Set("x-platform-error-code", "invalid")
				return c.String(400, fmt.Sprintf(`{"code":"invalid","message":"%v"}`, err.Error()))
			} else {
				return c.String(http.StatusOK, res)
			}
		}

	} else {
		res, err := service.RunE(c.Request().Context(), string(s))
		if err != nil {
			c.Response().Header().Set(echo.HeaderContentType, "application/json; charset=utf-8")
			c.Response().Header().Set("x-platform-error-code", "invalid")
			return c.String(400, fmt.Sprintf(`{"code":"invalid","message":"%v"}`, err.Error()))
		} else {
			return c.String(http.StatusOK, res)
		}
	}
}

func main() {

	port := flag.String("port", "8086", "API port")
	stdin := flag.Bool("stdin", false, "STDIN mode")
	cors := flag.Bool("cors", true, "API cors mode")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	inputString := ""

	if *stdin == true {

		for scanner.Scan() {
			inputString = inputString + "\n" + scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}

		buf, err := service.RunE(context.Background(), inputString)
		if err != nil {
			fmt.Fprintln(os.Stderr, "we have some error: ", err)
			return
		}

		fmt.Println(strings.Replace(buf, "\r\n", "\n", -1))

	} else {

		e := echo.New()
		e.HideBanner = true
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())

		if *cors == true {
			e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
			}))
		}

		e.GET("/", func(c echo.Context) error {
			return c.Blob(http.StatusOK, "text/html", static.PLAY)
		})
		e.GET("/favicon.ico", func(c echo.Context) error {
			return c.Blob(http.StatusOK, "image/x-icon", static.FAVICON)
		})

		e.GET("/hello", func(c echo.Context) error {
			return c.String(http.StatusOK, "|> FluxPIPE")
		})
		e.GET("/ping", func(c echo.Context) error {
			return c.String(204, "OK")
		})
		e.GET("/health", func(c echo.Context) error {
			return c.String(204, "OK")
		})
		e.POST("/api/v2/query", postQuery)
		e.POST("/query", postQuery)

		fmt.Println("|> FluxPIPE")
		e.Logger.Fatal(e.Start(":" + *port))
	}
}
