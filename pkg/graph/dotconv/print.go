// Copyright 2016-2017, Pulumi Corporation.  All rights reserved.

// Package dotconv converts a resource graph into its DOT digraph equivalent.  This is useful for integration with
// various visualization tools, like Graphviz.  Please see http://www.graphviz.org/content/dot-language for a thorough
// specification of the DOT file format.
package dotconv

import (
	"bufio"
	"fmt"
	"io"
	"strconv"

	"github.com/pulumi/pulumi/pkg/graph"
	"github.com/pulumi/pulumi/pkg/util/contract"
)

// Print prints a resource graph.
func Print(g graph.Graph, w io.Writer) error {
	// Allocate a new writer.  In general, we will ignore write errors throughout this function, for simplicity, opting
	// instead to return the result of flushing the buffer at the end, which is generally latching.
	b := bufio.NewWriter(w)

	// Print the graph header.
	if _, err := b.WriteString("strict digraph {\n"); err != nil {
		return err
	}

	// Initialize the frontier with unvisited graph vertices.
	queued := make(map[graph.Vertex]bool)
	frontier := make([]graph.Vertex, 0, len(g.Roots()))
	for _, root := range g.Roots() {
		to := root.To()
		queued[to] = true
		frontier = append(frontier, to)
	}

	// For now, we auto-generate IDs.
	// TODO[pulumi/pulumi#76]: use the object URNs instead, once we have them.
	c := 0
	ids := make(map[graph.Vertex]string)
	getID := func(v graph.Vertex) string {
		if id, has := ids[v]; has {
			return id
		}
		id := "Resource" + strconv.Itoa(c)
		c++
		ids[v] = id
		return id
	}

	// Now, until the frontier is empty, emit entries into the stream.
	indent := "    "
	emitted := make(map[graph.Vertex]bool)
	for len(frontier) > 0 {
		// Dequeue the head of the frontier.
		v := frontier[0]
		frontier = frontier[1:]
		contract.Assert(!emitted[v])
		emitted[v] = true

		// Get and lazily allocate the ID for this vertex.
		id := getID(v)

		// Print this vertex; first its "label" (type) and then its direct dependencies.
		// IDEA: consider serializing properties on the node also.
		if _, err := b.WriteString(fmt.Sprintf("%v%v", indent, id)); err != nil {
			return err
		}
		if label := v.Label(); label != "" {
			if _, err := b.WriteString(fmt.Sprintf(" [label=\"%v\"]", label)); err != nil {
				return err
			}
		}
		if _, err := b.WriteString(";\n"); err != nil {
			return err
		}

		// Now print out all dependencies as "ID -> {A ... Z}".
		outs := v.Outs()
		if len(outs) > 0 {
			base := fmt.Sprintf("%v%v", indent, id)
			// Print the ID of each dependency and, for those we haven't seen, add them to the frontier.
			for _, out := range outs {
				to := out.To()
				if _, err := b.WriteString(fmt.Sprintf("%s -> %s", base, getID(to))); err != nil {
					return err
				}

				if out.Color() != "" {
					if _, err := b.WriteString(fmt.Sprintf(" [color=\"%s\"]", out.Color())); err != nil {
						return err
					}
				}

				if _, err := b.WriteString(";\n"); err != nil {
					return err
				}

				if _, q := queued[to]; !q {
					queued[to] = true
					frontier = append(frontier, to)
				}
			}
		}
	}

	// Finish the graph.
	if _, err := b.WriteString("}\n"); err != nil {
		return err
	}
	return b.Flush()
}
