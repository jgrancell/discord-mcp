// Package resolve turns a Discord user or member name into an ID, never
// guessing across ambiguous matches.
package resolve

import (
	"context"
	"fmt"
	"strings"

	"github.com/jgrancell/discord-mcp/internal/discord"
)

// Named is a candidate a name resolved to (or could have resolved to).
type Named struct {
	ID   string
	Name string
}

// NotFound reports that no user matched the input name.
type NotFound struct {
	Kind      string
	Input     string
	Available []Named
}

func (e *NotFound) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("%s %q not found (none exist)", e.Kind, e.Input)
	}
	return fmt.Sprintf("%s %q not found; available: %s", e.Kind, e.Input, joinNames(e.Available))
}

// Ambiguous reports that multiple users matched. Tools surface this as a
// clarification prompt rather than picking one.
type Ambiguous struct {
	Kind       string
	Input      string
	Candidates []Named
}

func (e *Ambiguous) Error() string {
	return fmt.Sprintf("%q matches multiple %ss — which one? %s", e.Input, e.Kind, joinNames(e.Candidates))
}

func joinNames(ns []Named) string {
	names := make([]string, len(ns))
	for i, n := range ns {
		names[i] = n.Name
	}
	return strings.Join(names, ", ")
}

// Resolver resolves names against a live Discord client.
type Resolver struct {
	d *discord.Client
}

// New returns a Resolver over the given Discord client.
func New(d *discord.Client) *Resolver { return &Resolver{d: d} }

// match implements the resolution policy: exact case-insensitive match
// wins outright; otherwise substring matches are considered. Zero matches
// is a NotFound; multiple is an Ambiguous.
func match(kind, input string, all []Named) (Named, error) {
	needle := strings.ToLower(strings.TrimSpace(input))
	if needle == "" {
		return Named{}, &NotFound{Kind: kind, Input: input, Available: all}
	}

	var exact, partial []Named
	for _, n := range all {
		lower := strings.ToLower(n.Name)
		switch {
		case lower == needle:
			exact = append(exact, n)
		case strings.Contains(lower, needle):
			partial = append(partial, n)
		}
	}

	pick := exact
	if len(pick) == 0 {
		pick = partial
	}
	switch len(pick) {
	case 0:
		return Named{}, &NotFound{Kind: kind, Input: input, Available: all}
	case 1:
		return pick[0], nil
	default:
		return Named{}, &Ambiguous{Kind: kind, Input: input, Candidates: pick}
	}
}

// User resolves a username, display name, or guild nickname to a Discord
// user ID.
func (r *Resolver) User(ctx context.Context, guildID, name string) (Named, error) {
	members, err := r.d.SearchMembers(ctx, guildID, name)
	if err != nil {
		return Named{}, fmt.Errorf("search members: %w", err)
	}
	all := make([]Named, len(members))
	for i, m := range members {
		all[i] = Named{ID: m.ID, Name: m.DisplayName}
	}
	return match("user", name, all)
}
