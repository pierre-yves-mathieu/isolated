package cmd

import (
	"fmt"
	"strings"

	"lxc-dev-manager/internal/operations"

	"github.com/spf13/cobra"
)

// Parent command
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage images",
	Long:  `Manage container images (list, delete, rename).`,
}

// Alias: 'images' -> 'image list'
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images (alias for 'image list')",
	Long:  `List all local images. Alias for 'image list'.`,
	Args:  cobra.NoArgs,
	RunE:  runImageList,
}

// image list
var imageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local images",
	Long: `List all local images.

Example:
  lxc-dev-manager image list
  lxc-dev-manager image list --all`,
	Args: cobra.NoArgs,
	RunE: runImageList,
}

// image delete
var imageDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an image",
	Long: `Delete a local image by alias or fingerprint.
By default, asks for confirmation. Use --force to skip.

Example:
  lxc-dev-manager image delete my-base-image
  lxc-dev-manager image delete my-base-image --force`,
	Args: cobra.ExactArgs(1),
	RunE: runImageDelete,
}

// image rename
var imageRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename an image",
	Long: `Rename an image alias.

Example:
  lxc-dev-manager image rename my-base-image production-base`,
	Args: cobra.ExactArgs(2),
	RunE: runImageRename,
}

var imageListAll bool
var imageDeleteForce bool

func init() {
	// Add parent command
	rootCmd.AddCommand(imageCmd)

	// Add subcommands to image
	imageCmd.AddCommand(imageCreateCmd)
	imageCmd.AddCommand(imageListCmd)
	imageCmd.AddCommand(imageDeleteCmd)
	imageCmd.AddCommand(imageRenameCmd)

	// Add images alias at root level
	rootCmd.AddCommand(imagesCmd)

	// Flags
	imageListCmd.Flags().BoolVarP(&imageListAll, "all", "a", false, "Show all images including cached")
	imagesCmd.Flags().BoolVarP(&imageListAll, "all", "a", false, "Show all images including cached")
	imageDeleteCmd.Flags().BoolVarP(&imageDeleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runImageList(cmd *cobra.Command, args []string) error {
	// Use operations package to get image list
	images, err := operations.ListImages(imageListAll)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		if imageListAll {
			fmt.Println("No images found")
		} else {
			fmt.Println("No custom images found")
			fmt.Println("Use --all to show cached images")
		}
		return nil
	}

	// Print header
	fmt.Printf("%-25s %-14s %-10s %s\n", "ALIAS", "FINGERPRINT", "SIZE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 75))

	for _, img := range images {
		alias := img.Alias
		if alias == "" {
			alias = "-"
		}

		fp := img.Fingerprint
		if len(fp) > 12 {
			fp = fp[:12]
		}

		desc := img.Description
		if len(desc) > 25 {
			desc = desc[:22] + "..."
		}

		fmt.Printf("%-25s %-14s %-10s %s\n", alias, fp, img.Size, desc)
	}

	return nil
}

func runImageDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if exists
	if !operations.ImageExists(name) {
		return fmt.Errorf("image '%s' not found", name)
	}

	// Get image info for display
	images, _ := operations.ListImages(true)
	for _, img := range images {
		if img.Alias == name {
			fmt.Printf("\nImage: %s\n", name)
			fmt.Printf("  Size: %s\n", img.Size)
			if img.Description != "" {
				fmt.Printf("  Description: %s\n", img.Description)
			}
			fmt.Println()
			break
		}
	}

	// Ask for confirmation unless --force
	if !imageDeleteForce {
		if !confirmPrompt(fmt.Sprintf("Are you sure you want to delete image '%s'?", name)) {
			fmt.Println("Cancelled")
			return nil
		}
	}

	fmt.Printf("Deleting image '%s'...\n", name)
	if err := operations.DeleteImage(name); err != nil {
		return err
	}

	fmt.Printf("Image '%s' deleted\n", name)
	return nil
}

func runImageRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	// Check if old exists
	if !operations.ImageExists(oldName) {
		return fmt.Errorf("image '%s' not found", oldName)
	}

	// Check if new already exists
	if operations.ImageExists(newName) {
		return fmt.Errorf("image '%s' already exists", newName)
	}

	fmt.Printf("Renaming image '%s' → '%s'...\n", oldName, newName)
	if err := operations.RenameImage(oldName, newName); err != nil {
		return err
	}

	fmt.Printf("Image renamed: %s → %s\n", oldName, newName)
	return nil
}
