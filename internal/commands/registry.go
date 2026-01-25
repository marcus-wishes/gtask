package commands

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds registered commands.
type Registry struct {
	mu   sync.RWMutex
	cmds map[string]Command // name and aliases map to command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		cmds: make(map[string]Command),
	}
}

// Register adds a command to the registry.
// Returns an error if the name or any alias is already registered.
func (r *Registry) Register(c Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := c.Name()
	if _, exists := r.cmds[name]; exists {
		return fmt.Errorf("command already registered: %s", name)
	}

	for _, alias := range c.Aliases() {
		if _, exists := r.cmds[alias]; exists {
			return fmt.Errorf("command alias already registered: %s", alias)
		}
	}

	r.cmds[name] = c
	for _, alias := range c.Aliases() {
		r.cmds[alias] = c
	}

	return nil
}

// Find looks up a command by name or alias.
func (r *Registry) Find(name string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.cmds[name]
	return cmd, ok
}

// All returns all unique commands sorted by name.
func (r *Registry) All() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect unique commands by primary name
	seen := make(map[string]Command)
	for _, cmd := range r.cmds {
		seen[cmd.Name()] = cmd
	}

	// Sort by name
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Command, len(names))
	for i, name := range names {
		result[i] = seen[name]
	}
	return result
}

// DefaultRegistry is the global command registry.
var DefaultRegistry = NewRegistry()

// Register adds a command to the default registry.
func Register(c Command) {
	if err := DefaultRegistry.Register(c); err != nil {
		panic(err)
	}
}
