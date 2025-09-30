package cmd

import (
	"fmt"
	"html/template"
	"net/http"
	"taskflow/internal/config"
	"taskflow/internal/storage"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a web interface for task management",
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
			storagePath := config.GetStoragePath()
			s, err := storage.NewStorage(storagePath)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creating storage: %v", err), http.StatusInternalServerError)
				return
			}

			tasks, err := s.ReadTasks()
			if err != nil {
				http.Error(w, fmt.Sprintf("Error reading tasks: %v", err), http.StatusInternalServerError)
				return
			}

			tmpl, err := template.New("tasks").Parse(`
				<h1>Task List</h1>
				<ul>
					{{range .}}
						<li>{{.Title}} - {{if .Completed}}Done{{else}}Pending{{end}}</li>
					{{else}}
						<li>No tasks found.</li>
					{{end}}
				</ul>
			`)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error parsing template: %v", err), http.StatusInternalServerError)
				return
			}

			tmpl.Execute(w, tasks)
		})

		port := ":8081"
		fmt.Printf("Starting web server on http://localhost%s/tasks\n", port)
		http.ListenAndServe(port, nil)
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
}