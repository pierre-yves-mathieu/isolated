package lxcmgr

import (
	"io"

	"lxc-dev-manager/internal/operations"
)

// ListImages returns all local images
func ListImages(all bool) ([]ImageInfo, error) {
	images, err := operations.ListImages(all)
	if err != nil {
		return nil, err
	}

	var result []ImageInfo
	for _, img := range images {
		result = append(result, ImageInfo{
			Alias:       img.Alias,
			Fingerprint: img.Fingerprint,
			Size:        img.Size,
			Description: img.Description,
		})
	}

	return result, nil
}

// CreateImage creates an image from a container
func (c *Client) CreateImage(container, imageName string) error {
	return c.CreateImageWithProgress(container, imageName, nil, nil)
}

// CreateImageWithProgress creates an image from a container with progress output
func (c *Client) CreateImageWithProgress(container, imageName string, stdout, stderr io.Writer) error {
	return operations.CreateImage(c.cfg, container, imageName, stdout, stderr)
}

// DeleteImage deletes an image by alias
func DeleteImage(name string) error {
	return operations.DeleteImage(name)
}

// RenameImage renames an image
func RenameImage(oldName, newName string) error {
	return operations.RenameImage(oldName, newName)
}

// ImageExists checks if an image exists
func ImageExists(name string) bool {
	return operations.ImageExists(name)
}
