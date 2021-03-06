package hoff

import (
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
)

// NodeSystem is a system to configure workflow between action nodes, or decision nodes.
// The nodes are linked between them by link and join mode options.
// An activated Node system will be walked throw Follow and Ancestors functions
type NodeSystem struct {
	activated      bool
	nodes          []Node
	nodesJoinModes map[Node]JoinMode
	links          []nodeLink

	initialNodes       []Node
	followingNodesTree map[Node]map[*bool][]Node
	ancestorsNodesTree map[Node]map[*bool][]Node
}

// NewNodeSystem create an empty Node system
// who need to be valid and activated in order to be used.
func NewNodeSystem() *NodeSystem {
	return &NodeSystem{
		activated:          false,
		nodes:              make([]Node, 0),
		links:              make([]nodeLink, 0),
		nodesJoinModes:     make(map[Node]JoinMode),
		initialNodes:       make([]Node, 0),
		followingNodesTree: make(map[Node]map[*bool][]Node),
		ancestorsNodesTree: make(map[Node]map[*bool][]Node),
	}
}

// Equal validate the two NodeSystem are equals.
func (s *NodeSystem) Equal(o *NodeSystem) bool {
	return cmp.Equal(s.activated, o.activated) && cmp.Equal(s.nodes, o.nodes, NodeComparator) && cmp.Equal(s.nodesJoinModes, o.nodesJoinModes) && cmp.Equal(s.links, o.links, nodeLinkComparator)
}

// AddNode add a node to the system before activation.
func (s *NodeSystem) AddNode(n Node) (bool, error) {
	if s.activated {
		return false, errors.New("can't add node, node system is freeze due to activation")
	}
	s.nodes = append(s.nodes, n)
	return true, nil
}

// ConfigureJoinModeOnNode configure the join mode of a node into the system before activation.
func (s *NodeSystem) ConfigureJoinModeOnNode(n Node, m JoinMode) (bool, error) {
	if s.activated {
		return false, errors.New("can't add node join mode, node system is freeze due to activation")
	}
	s.nodesJoinModes[n] = m
	return true, nil
}

// AddLink add a link from a node to another node into the system before activation.
func (s *NodeSystem) AddLink(from, to Node) (bool, error) {
	return s.addLink(from, to, nil)
}

// AddLinkOnBranch add a link from a node (on a specific branch) to another node into the system before activation.
func (s *NodeSystem) AddLinkOnBranch(from, to Node, branch bool) (bool, error) {
	return s.addLink(from, to, &branch)
}

// IsValid check if the configuration of the node system is valid based on checks.
// Check for decision node with any node links as from,
// check for cyclic redundancy in node links,
// check for undeclared node used in node links,
// check for multiple declaration of same node instance.
func (s *NodeSystem) IsValid() (bool, []error) {
	errors := make([]error, 0)
	errors = append(errors, checkForOrphanMultiBranchesNode(s)...)
	errors = append(errors, checkForCyclicRedundancyInNodeLinks(s)...)
	errors = append(errors, checkForUndeclaredNodeInNodeLink(s)...)
	errors = append(errors, checkForMultipleInstanceOfSameNode(s)...)
	errors = append(errors, checkForMultipleLinksToNodeWithoutJoinMode(s)...)

	if len(errors) == 0 {
		return true, nil
	}
	return false, errors
}

// Activate prepare the node system to be used.
// In order to activate it, the node system must be valid.
// Once activated, the initial nodes, following nodes, and ancestors nodes will be accessibles.
func (s *NodeSystem) Activate() error {
	if s.activated {
		return nil
	}

	validity, _ := s.IsValid()
	if !validity {
		return errors.New("can't activate a unvalidated node system")
	}

	initialNodes := make([]Node, 0)
	followingNodesTree := make(map[Node]map[*bool][]Node)
	ancestorsNodesTree := make(map[Node]map[*bool][]Node)

	toNodes := make([]Node, 0)
	for _, link := range s.links {
		followingNodesTreeOnBranch, foundNode := followingNodesTree[link.From]
		if !foundNode {
			followingNodesTree[link.From] = make(map[*bool][]Node)
			followingNodesTreeOnBranch = followingNodesTree[link.From]
		}
		followingNodesTreeOnBranch[link.Branch] = append(followingNodesTreeOnBranch[link.Branch], link.To)

		ancestorsNodesTreeOnBranch, foundNode := ancestorsNodesTree[link.To]
		if !foundNode {
			ancestorsNodesTree[link.To] = make(map[*bool][]Node)
			ancestorsNodesTreeOnBranch = ancestorsNodesTree[link.To]
		}
		ancestorsNodesTreeOnBranch[link.Branch] = append(ancestorsNodesTreeOnBranch[link.Branch], link.From)

		toNodes = append(toNodes, link.To)
	}

	for _, node := range s.nodes {
		isInitialNode := true
		for _, toNode := range toNodes {
			if node == toNode {
				isInitialNode = false
				break
			}
		}
		if isInitialNode {
			initialNodes = append(initialNodes, node)
		}
	}

	s.initialNodes = initialNodes
	s.followingNodesTree = followingNodesTree
	s.ancestorsNodesTree = ancestorsNodesTree

	s.activated = true
	return nil
}

// JoinModeOfNode get the configured join mode of a node
func (s *NodeSystem) JoinModeOfNode(n Node) JoinMode {
	mode, foundMode := s.nodesJoinModes[n]
	if foundMode {
		return mode
	}
	return JoinNone
}

// InitialNodes get the initial nodes
func (s *NodeSystem) InitialNodes() []Node {
	return s.initialNodes
}

// IsActivated give the activation state of the node system.
// Only true if the node system is valid and have run the activate function without errors.
func (s *NodeSystem) IsActivated() bool {
	return s.activated
}

// Follow get the set of nodes accessible from a specific node and one of its branch after activation.
func (s *NodeSystem) Follow(n Node, branch *bool) ([]Node, error) {
	if !s.activated {
		return nil, errors.New("can't follow a node if system is not activated")
	}
	links, foundLinks := s.followingNodesTree[n]
	if foundLinks {
		nodes, foundNodes := links[branch]
		if foundNodes {
			return nodes, nil
		}
	}
	return nil, nil
}

// Ancestors get the set of nodes who access using one of their branch to a specific node after activation.
func (s *NodeSystem) Ancestors(n Node, branch *bool) ([]Node, error) {
	if !s.activated {
		return nil, errors.New("can't get ancestors of a node if system is not activated")
	}
	links, foundLinks := s.ancestorsNodesTree[n]
	if foundLinks {
		nodes, foundNodes := links[branch]
		if foundNodes {
			return nodes, nil
		}
	}
	return nil, nil
}

func (s *NodeSystem) addLink(from, to Node, branch *bool) (bool, error) {
	if s.activated {
		return false, errors.New("can't add branch link, node system is freeze due to activation")
	}

	if from == nil {
		return false, fmt.Errorf("can't have missing 'from' attribute")
	}

	if branch == nil && from.DecideCapability() {
		return false, fmt.Errorf("can't have missing branch")
	}

	if branch != nil && !from.DecideCapability() {
		return false, fmt.Errorf("can't have not needed branch")
	}

	if to == nil {
		return false, fmt.Errorf("can't have missing 'to' attribute")
	}

	if from == to {
		return false, fmt.Errorf("can't have link on from and to the same node")
	}

	if branch == nil {
		s.links = append(s.links, newNodeLink(from, to))
	} else {
		s.links = append(s.links, newNodeLinkOnBranch(from, to, *branch))
	}
	return true, nil
}

func (s *NodeSystem) haveNode(n Node) bool {
	for _, node := range s.nodes {
		if node == n {
			return true
		}
	}
	return false
}

func checkForOrphanMultiBranchesNode(s *NodeSystem) []error {
	errors := make([]error, 0)
	for _, node := range s.nodes {
		if node.DecideCapability() {
			noLink := true
			for _, link := range s.links {
				if link.From == node {
					noLink = false
					break
				}
			}
			if noLink {
				errors = append(errors, fmt.Errorf("can't have decision node without link from it: %+v", node))
			}
		}
	}
	return errors
}

func checkForCyclicRedundancyInNodeLinks(s *NodeSystem) []error {
	errors := make([]error, 0)
	cycles := make([][]nodeLink, 0)
	for _, node := range s.nodes {
		possibleCycles := findCycle(s, node, node, nil)
		cycles = append(cycles, possibleCycles...)
	}

	nodeLinkSliceComparator := cmp.Comparer(func(x, y []nodeLink) bool {
		sameLinkCount := 0
		for _, xItem := range x {
			foundIt := false
			for _, yItem := range y {
				if cmp.Equal(xItem, yItem, nodeLinkComparator) {
					foundIt = true
					break
				}
			}
			if foundIt {
				sameLinkCount++
			}
		}
		return sameLinkCount == len(x)
	})

	trimmedCycles := make([][]nodeLink, 0)
	for _, cycle := range cycles {
		alreadyTrimmed := false
		for _, trimmedCycle := range trimmedCycles {
			if cmp.Equal(cycle, trimmedCycle, nodeLinkSliceComparator) {
				alreadyTrimmed = true
			}
		}
		if !alreadyTrimmed {
			trimmedCycles = append(trimmedCycles, cycle)
		}
	}

	for _, cycle := range trimmedCycles {
		errors = append(errors, fmt.Errorf("Can't have cycle in links between nodes: %+v", cycle))
	}
	return errors
}

func findCycle(s *NodeSystem, topNode, currentNode Node, walkednodeLinks []nodeLink) [][]nodeLink {
	if walkednodeLinks != nil && len(walkednodeLinks) > 0 {
		if topNode == currentNode {
			return [][]nodeLink{walkednodeLinks}
		}
		for _, link := range walkednodeLinks {
			if currentNode == link.From {
				return [][]nodeLink{}
			}
		}
	}
	var selectedLinks []nodeLink
	for _, link := range s.links {
		if link.From == currentNode {
			selectedLinks = append(selectedLinks, link)
		}
	}

	if len(selectedLinks) == 0 {
		return nil
	}

	cycles := make([][]nodeLink, 0)
	for _, link := range selectedLinks {
		newWalkednodeLinks := append(walkednodeLinks, link)
		linkCycles := findCycle(s, topNode, link.To, newWalkednodeLinks)
		cycles = append(cycles, linkCycles...)
	}
	return cycles
}

func checkForUndeclaredNodeInNodeLink(s *NodeSystem) []error {
	errors := make([]error, 0)
	for _, link := range s.links {
		if link.From != nil && !s.haveNode(link.From) {
			errors = append(errors, fmt.Errorf("can't have undeclared node '%+v' as 'from' in branch link %+v", link.From, link))
		}
		if link.To != nil && !s.haveNode(link.To) {
			errors = append(errors, fmt.Errorf("can't have undeclared node '%+v' as 'to' in branch link %+v", link.To, link))
		}
	}
	return errors
}

func checkForMultipleInstanceOfSameNode(s *NodeSystem) []error {
	errors := make([]error, 0)
	count := make(map[Node]int)
	for i := 0; i < len(s.nodes); i++ {
		for j := 0; j < len(s.nodes); j++ {
			if i != j && s.nodes[i] == s.nodes[j] {
				count[s.nodes[i]]++
			}
		}
	}
	for n, c := range count {
		if c > 1 {
			errors = append(errors, fmt.Errorf("can't have multiple instances (%v) of the same node: %+v", c, n))
		}
	}
	return errors
}

func checkForMultipleLinksToNodeWithoutJoinMode(s *NodeSystem) []error {
	errors := make([]error, 0)
	count := make(map[Node]int)
	for _, link := range s.links {
		count[link.To]++
	}
	for n, c := range count {
		if c > 1 && s.JoinModeOfNode(n) == JoinNone {
			errors = append(errors, fmt.Errorf("can't have multiple links (%v) to the same node: %+v without join mode", c, n))
		}
	}
	return errors
}
