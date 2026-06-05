package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"taskmanager/internal/db"
	"taskmanager/internal/db/repository"

	"github.com/spf13/cobra"
)

var (
	dbConn *sql.DB
	repo   *repository.TaskRepository
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "tasks",
	Short: "A simple CLI task manager backed by MySQL",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg := db.LoadConfig()
		conn, err := db.Connect(cfg)
		if err != nil {
			return fmt.Errorf("DB connection failed: %w", err)
		}
		dbConn = conn
		repo = repository.NewTaskRepository(dbConn)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if dbConn != nil {
			dbConn.Close()
		}
	},
}

// add

var addDesc string

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new task",
	Example: `  tasks add "Buy groceries"
  tasks add "Write report" --desc "Q2 financial report"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		task, err := repo.Create(args[0], addDesc)
		if err != nil {
			return err
		}
		fmt.Printf("Task #%d created: %s\n", task.ID, task.Title)
		return nil
	},
}

// list

var listFilter string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Example: `  tasks list
  tasks list --filter todo
  tasks list --filter done`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, err := repo.List(listFilter)
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tCREATED")
		fmt.Fprintln(w, "--\t-----\t------\t-------")
		for _, t := range tasks {
			status := "[ ] todo"
			if t.Status == "done" {
				status = "[✓] done"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
				t.ID, t.Title, status,
				t.CreatedAt.Format(time.RFC822),
			)
		}
		w.Flush()
		return nil
	},
}

// get

var getCmd = &cobra.Command{
	Use:     "get <id>",
	Short:   "Show details of a task",
	Example: `  tasks get 3`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		t, err := repo.GetByID(id)
		if err != nil {
			return err
		}

		fmt.Printf("\nTask #%d\n", t.ID)
		fmt.Printf("  Title      : %s\n", t.Title)
		fmt.Printf("  Description: %s\n", t.Description)
		fmt.Printf("  Status     : %s\n", t.Status)
		fmt.Printf("  Created    : %s\n", t.CreatedAt.Format(time.RFC1123))
		fmt.Printf("  Updated    : %s\n\n", t.UpdatedAt.Format(time.RFC1123))
		return nil
	},
}

// update

var (
	updateTitle string
	updateDesc  string
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task's title or description",
	Example: `  tasks update 3 --title "New title"
  tasks update 3 --desc "New description"
  tasks update 3 --title "New title" --desc "New description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if updateTitle == "" && updateDesc == "" {
			return fmt.Errorf("provide at least --title or --desc to update")
		}
		task, err := repo.Update(id, updateTitle, updateDesc)
		if err != nil {
			return err
		}
		fmt.Printf("Task #%d updated: %s\n", task.ID, task.Title)
		return nil
	},
}

var doneCmd = &cobra.Command{
	Use:     "done <id>",
	Short:   "Mark a task as done",
	Example: `  tasks done 3`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if err := repo.MarkDone(id); err != nil {
			return err
		}
		fmt.Printf("Task #%d marked as done.\n", id)
		return nil
	},
}

// delete

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a task permanently",
	Example: `  tasks delete 3`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		if err := repo.Delete(id); err != nil {
			return err
		}
		fmt.Printf("Task #%d deleted.\n", id)
		return nil
	},
}

func Execute() {
	addCmd.Flags().StringVarP(&addDesc, "desc", "d", "", "Task description")

	listCmd.Flags().StringVarP(&listFilter, "filter", "f", "", "Filter by status: todo | done")

	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "New title")
	updateCmd.Flags().StringVarP(&updateDesc, "desc", "d", "", "New description")

	rootCmd.AddCommand(addCmd, listCmd, getCmd, updateCmd, doneCmd, deleteCmd)

	// MDF operational table commands
	RegisterMDFCommands()

	// RabbitMQ commands
	RegisterMQCommands()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q — must be a positive integer", s)
	}
	return id, nil
}
