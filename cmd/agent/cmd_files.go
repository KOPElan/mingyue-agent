package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func filesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "File management operations",
		Long:  "Perform file management operations such as list, info, mkdir, delete, copy, move, etc.",
	}

	cmd.AddCommand(filesListCmd())
	cmd.AddCommand(filesInfoCmd())
	cmd.AddCommand(filesMkdirCmd())
	cmd.AddCommand(filesDeleteCmd())
	cmd.AddCommand(filesCopyCmd())
	cmd.AddCommand(filesMoveCmd())

	return cmd
}

func filesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <path>",
		Short: "List files in a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			path := args[0]

			resp, err := client.Get(fmt.Sprintf("/api/v1/files/list?path=%s", url.QueryEscape(path)))
			if err != nil {
				return err
			}

			var files []struct {
				Name        string    `json:"name"`
				Path        string    `json:"path"`
				Size        int64     `json:"size"`
				ModTime     time.Time `json:"mod_time"`
				IsDir       bool      `json:"is_dir"`
				IsSymlink   bool      `json:"is_symlink"`
				Permissions string    `json:"permissions"`
			}

			if err := json.Unmarshal(resp.Data, &files); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tPERMISSIONS\tSIZE\tMODIFIED\tNAME")
			for _, f := range files {
				ftype := "file"
				if f.IsDir {
					ftype = "dir"
				} else if f.IsSymlink {
					ftype = "link"
				}
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
					ftype, f.Permissions, f.Size, f.ModTime.Format("2006-01-02 15:04:05"), f.Name)
			}
			w.Flush()

			return nil
		},
	}
}

func filesInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <path>",
		Short: "Get file or directory information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			path := args[0]

			resp, err := client.Get(fmt.Sprintf("/api/v1/files/info?path=%s", url.QueryEscape(path)))
			if err != nil {
				return err
			}

			var info struct {
				Name        string    `json:"name"`
				Path        string    `json:"path"`
				Size        int64     `json:"size"`
				Mode        uint32    `json:"mode"`
				ModTime     time.Time `json:"mod_time"`
				IsDir       bool      `json:"is_dir"`
				IsSymlink   bool      `json:"is_symlink"`
				Owner       uint32    `json:"owner"`
				Group       uint32    `json:"group"`
				Permissions string    `json:"permissions"`
			}

			if err := json.Unmarshal(resp.Data, &info); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Name:        %s\n", info.Name)
			fmt.Printf("Path:        %s\n", info.Path)
			fmt.Printf("Type:        %s\n", func() string {
				if info.IsDir {
					return "directory"
				} else if info.IsSymlink {
					return "symlink"
				}
				return "file"
			}())
			fmt.Printf("Size:        %d bytes\n", info.Size)
			fmt.Printf("Permissions: %s\n", info.Permissions)
			fmt.Printf("Owner:       %d\n", info.Owner)
			fmt.Printf("Group:       %d\n", info.Group)
			fmt.Printf("Modified:    %s\n", info.ModTime.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
}

func filesMkdirCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mkdir <path>",
		Short: "Create a new directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			path := args[0]

			_, err := client.Post("/api/v1/files/mkdir", map[string]string{"path": path})
			if err != nil {
				return err
			}

			fmt.Printf("Directory created: %s\n", path)
			return nil
		},
	}
}

func filesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a file or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			path := args[0]

			_, err := client.Post("/api/v1/files/delete", map[string]string{"path": path})
			if err != nil {
				return err
			}

			fmt.Printf("Deleted: %s\n", path)
			return nil
		},
	}
}

func filesCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <source> <destination>",
		Short: "Copy a file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			src := args[0]
			dst := args[1]

			_, err := client.Post("/api/v1/files/copy", map[string]string{
				"src_path": src,
				"dst_path": dst,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Copied %s -> %s\n", src, dst)
			return nil
		},
	}
}

func filesMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <source> <destination>",
		Short: "Move a file or directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			src := args[0]
			dst := args[1]

			_, err := client.Post("/api/v1/files/move", map[string]string{
				"src_path": src,
				"dst_path": dst,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Moved %s -> %s\n", src, dst)
			return nil
		},
	}
}
