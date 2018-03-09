package dockerfile

import (
	"fmt"
	"strings"

	"github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
)

// FindAll returns the indices of all children of node such that
// node.Children[i].Value == cmd. Valid values for cmd are defined in the
// package github.com/docker/docker/builder/dockerfile/command.
func FindAll(node *parser.Node, cmd string) []int {
	if node == nil {
		return nil
	}
	var indices []int
	for i, child := range node.Children {
		if child != nil && child.Value == cmd {
			indices = append(indices, i)
		}
	}
	return indices
}

// InsertInstructions inserts instructions starting from the pos-th child of
// node, moving other children as necessary. The instructions should be valid
// Dockerfile instructions. InsertInstructions mutates node in-place, and the
// final state of node is equivalent to what parser.Parse would return if the
// original Dockerfile represented by node contained the instructions at the
// specified position pos. If the returned error is non-nil, node is guaranteed
// to be unchanged.
func InsertInstructions(node *parser.Node, pos int, instructions string) error {
	if node == nil {
		return fmt.Errorf("cannot insert instructions in a nil node")
	}
	if pos < 0 || pos > len(node.Children) {
		return fmt.Errorf("pos %d out of range [0, %d]", pos, len(node.Children)-1)
	}
	newChild, err := parser.Parse(strings.NewReader(instructions))
	if err != nil {
		return err
	}
	// InsertVector pattern (https://github.com/golang/go/wiki/SliceTricks)
	node.Children = append(node.Children[:pos], append(newChild.AST.Children, node.Children[pos:]...)...)
	return nil
}

// LastBaseImage takes a Dockerfile root node and returns the base image
// declared in the last FROM instruction.
func LastBaseImage(node *parser.Node) string {
	baseImages := baseImages(node)
	if len(baseImages) == 0 {
		return ""
	}
	return baseImages[len(baseImages)-1]
}

// baseImages takes a Dockerfile root node and returns a list of all base images
// declared in the Dockerfile. Each base image is the argument of a FROM
// instruction.
func baseImages(node *parser.Node) []string {
	var images []string
	for _, pos := range FindAll(node, command.From) {
		images = append(images, nextValues(node.Children[pos])...)
	}
	return images
}

// LastExposedPorts takes a Dockerfile root node and returns a list of ports
// exposed in the last image built by the Dockerfile, i.e., only the EXPOSE
// instructions after the last FROM instruction are considered.
func LastExposedPorts(node *parser.Node) []string {
	exposedPorts := exposedPorts(node)
	if len(exposedPorts) == 0 {
		return nil
	}
	return exposedPorts[len(exposedPorts)-1]
}

// exposedPorts takes a Dockerfile root node and returns a list of all ports
// exposed in the Dockerfile, grouped by images that this Dockerfile produces.
// The number of port lists returned is the number of images produced by this
// Dockerfile, which is the same as the number of FROM instructions.
func exposedPorts(node *parser.Node) [][]string {
	var allPorts [][]string
	var ports []string
	froms := FindAll(node, command.From)
	exposes := FindAll(node, command.Expose)
	for i, j := len(froms)-1, len(exposes)-1; i >= 0; i-- {
		for ; j >= 0 && exposes[j] > froms[i]; j-- {
			ports = append(nextValues(node.Children[exposes[j]]), ports...)
		}
		allPorts = append([][]string{ports}, allPorts...)
		ports = nil
	}
	return allPorts
}

// nextValues returns a slice of values from the next nodes following node. This
// roughly translates to the arguments to the Docker builder instruction
// represented by node.
func nextValues(node *parser.Node) []string {
	if node == nil {
		return nil
	}
	var values []string
	for next := node.Next; next != nil; next = next.Next {
		values = append(values, next.Value)
	}
	return values
}
