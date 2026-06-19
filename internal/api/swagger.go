package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/example/purchase-api/docs"
)

// @title Purchase API
// @version 1.0.0
// @description RESTful API for managing purchase transactions with multi-currency support
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.basic BasicAuth

// swaggerHandler serves the Swagger UI HTML page
func swaggerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Purchase API - Swagger UI</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@3/swagger-ui.css">
	<style>
		html {
			box-sizing: border-box;
			overflow: -moz-scrollbars-vertical;
			overflow-y: scroll;
		}
		*, *:before, *:after {
			box-sizing: inherit;
		}
		body {
			margin: 0;
			padding: 0;
		}
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
	<script>
		window.onload = function() {
			// Configure Swagger UI
			window.ui = SwaggerUIBundle({
				url: "/docs/swagger.json",
				dom_id: '#swagger-ui',
				presets: [
					SwaggerUIBundle.presets.apis
				],
				deepLinking: true
			})
		}
	</script>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, html)
	}
}

// swaggerJSONHandler serves the Swagger JSON specification
func swaggerJSONHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to read the generated swagger.json
		possiblePaths := []string{
			"docs/swagger.json",
			"./docs/swagger.json",
			"../docs/swagger.json",
			"../../docs/swagger.json",
		}

		var specData []byte
		var err error

		for _, path := range possiblePaths {
			specData, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			// Try absolute path
			cwd, _ := os.Getwd()
			absPath := filepath.Join(cwd, "docs", "swagger.json")
			specData, err = os.ReadFile(absPath)
		}

		if len(specData) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"swagger spec not generated. Run 'swag init -g cmd/purchase-api/main.go' to generate it"}`)
			return
		}

		// Parse to verify it's valid JSON and re-encode
		var spec map[string]interface{}
		if err := json.Unmarshal(specData, &spec); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(spec)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"failed to parse swagger spec"}`)
		}
	}
}

// swaggerYAMLHandler serves the Swagger YAML specification
func swaggerYAMLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to read the generated swagger.yaml
		possiblePaths := []string{
			"docs/swagger.yaml",
			"./docs/swagger.yaml",
			"../docs/swagger.yaml",
			"../../docs/swagger.yaml",
		}

		var specData []byte
		var err error

		for _, path := range possiblePaths {
			specData, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			// Try absolute path
			cwd, _ := os.Getwd()
			absPath := filepath.Join(cwd, "docs", "swagger.yaml")
			specData, err = os.ReadFile(absPath)
		}

		if len(specData) == 0 {
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "# Swagger spec not found. Run 'swag init -g cmd/purchase-api/main.go' to generate it\n")
			return
		}

		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		w.Write(specData)
	}
}
