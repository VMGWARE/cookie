package server

import (
	"fmt"

	"github.com/MarceloPetrucio/go-scalar-api-reference"
)

func (s *Server) swaggerReference(w *responseWriter, r *request) error {
	htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
		// TODO: Load the swagger.json file from the server instead of the local file
		SpecURL: "./docs/swagger.json",
		CustomOptions: scalar.CustomOptions{
			PageTitle: "Camphouse - API Reference",
		},
		DarkMode: true,
	})

	if err != nil {
		fmt.Printf("%v", err)
	}

	w.Header().Set("Content-Type", "text/html")

	fmt.Fprintln(w, htmlContent)

	return nil
}
