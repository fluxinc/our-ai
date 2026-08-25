package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/fluxinc/my-cli/internal/customers"
)

const customerIdentityProjectionVersion = 1

type customerIdentityProjection struct {
	SchemaVersion int                      `json:"schema_version"`
	Sources       []customerIdentitySource `json:"sources"`
	Customers     []customerIdentity       `json:"customers"`
}

type customerIdentitySource struct {
	Manifest  string `json:"manifest"`
	Workspace string `json:"workspace"`
	Revision  string `json:"revision"`
	Freshness string `json:"freshness"`
}

type customerIdentity struct {
	ID              string   `json:"id"`
	Domain          string   `json:"domain,omitempty"`
	DomainConfirmed bool     `json:"domain_confirmed,omitempty"`
	Aliases         []string `json:"aliases,omitempty"`
}

func buildCustomerIdentityProjection(roots []customers.Root, found []customers.Customer) (customerIdentityProjection, error) {
	if len(roots) == 0 {
		return customerIdentityProjection{}, structuredCommandError{
			code: "customer_identity_source_unavailable", message: "no customer identity source is available",
			remediation: "run my setup or pass --manifest, --workspace, or --umbrella",
		}
	}
	projection := customerIdentityProjection{SchemaVersion: customerIdentityProjectionVersion}
	seenSources := map[string]bool{}
	for _, root := range roots {
		key := customerIdentityManifest(root.Manifest) + "\x00" + root.Workspace + "\x00" + root.Path
		if seenSources[key] {
			continue
		}
		seenSources[key] = true
		source, err := inspectCustomerIdentitySource(root)
		if err != nil {
			return customerIdentityProjection{}, err
		}
		projection.Sources = append(projection.Sources, source)
	}
	sort.Slice(projection.Sources, func(i, j int) bool {
		if projection.Sources[i].Manifest != projection.Sources[j].Manifest {
			return projection.Sources[i].Manifest < projection.Sources[j].Manifest
		}
		if projection.Sources[i].Workspace != projection.Sources[j].Workspace {
			return projection.Sources[i].Workspace < projection.Sources[j].Workspace
		}
		if projection.Sources[i].Revision != projection.Sources[j].Revision {
			return projection.Sources[i].Revision < projection.Sources[j].Revision
		}
		return projection.Sources[i].Freshness < projection.Sources[j].Freshness
	})
	for _, customer := range found {
		aliases := append([]string(nil), customer.Aliases...)
		sort.Strings(aliases)
		projection.Customers = append(projection.Customers, customerIdentity{
			ID: customer.ID, Domain: customer.Domain, DomainConfirmed: customer.DomainConfirmed, Aliases: aliases,
		})
	}
	sort.Slice(projection.Customers, func(i, j int) bool {
		return projection.Customers[i].ID < projection.Customers[j].ID
	})
	return projection, nil
}

func inspectCustomerIdentitySource(root customers.Root) (customerIdentitySource, error) {
	git := func(args ...string) (string, error) {
		cmd := exec.Command("git", append([]string{"-C", root.Path}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%s", commandMessage(out, err))
		}
		return strings.TrimSpace(string(out)), nil
	}
	fail := func(code, message, remediation string) (customerIdentitySource, error) {
		return customerIdentitySource{}, structuredCommandError{code: code, message: message, remediation: remediation}
	}
	revision, err := git("rev-parse", "HEAD")
	if err != nil {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("customer identity source %s/%s is not a readable Git checkout: %v", root.Manifest, root.Workspace, err), "run my mounts sync or my setup")
	}
	dirty, err := git("status", "--porcelain=v1", "--", "customers")
	if err != nil {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("cannot inspect customer identity source %s/%s: %v", root.Manifest, root.Workspace, err), "run my doctor")
	}
	if dirty != "" {
		return fail("customer_identity_source_dirty", fmt.Sprintf("customer identity source %s/%s has uncommitted customer records; commit or discard them before exporting a projection", root.Manifest, root.Workspace), "git -C "+shellQuote(root.Path)+" status --short -- customers")
	}
	source := customerIdentitySource{Manifest: customerIdentityManifest(root.Manifest), Workspace: root.Workspace, Revision: revision, Freshness: "local"}
	if _, err := git("remote", "get-url", "origin"); err != nil {
		return source, nil
	}
	if _, err := git("fetch", "origin"); err != nil {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("cannot refresh customer identity source %s/%s: %v", root.Manifest, root.Workspace, err), "run my sync and retry")
	}
	branch, err := git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "HEAD" {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("customer identity source %s/%s is not on a branch", root.Manifest, root.Workspace), "run my doctor")
	}
	upstream, err := git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil || upstream == "" {
		upstream = "origin/" + branch
		if _, verifyErr := git("rev-parse", "--verify", upstream); verifyErr != nil {
			return fail("customer_identity_source_unavailable", fmt.Sprintf("customer identity source %s/%s has no remote branch for %s", root.Manifest, root.Workspace, branch), "run my sync and retry")
		}
	}
	counts, err := git("rev-list", "--left-right", "--count", upstream+"...HEAD")
	if err != nil {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("cannot compare customer identity source %s/%s with its remote: %v", root.Manifest, root.Workspace, err), "run my doctor")
	}
	fields := strings.Fields(counts)
	if len(fields) != 2 {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("cannot parse freshness for customer identity source %s/%s", root.Manifest, root.Workspace), "run my doctor")
	}
	behind, behindErr := strconv.Atoi(fields[0])
	ahead, aheadErr := strconv.Atoi(fields[1])
	if behindErr != nil || aheadErr != nil {
		return fail("customer_identity_source_unavailable", fmt.Sprintf("cannot parse freshness for customer identity source %s/%s", root.Manifest, root.Workspace), "run my doctor")
	}
	if ahead != 0 || behind != 0 {
		return fail("customer_identity_source_stale", fmt.Sprintf("customer identity source %s/%s is not synchronized with %s (ahead %d, behind %d)", root.Manifest, root.Workspace, upstream, ahead, behind), "run my sync and retry")
	}
	source.Freshness = "current"
	return source, nil
}

func customerIdentityManifest(value string) string {
	parts := strings.Split(value, ":")
	if len(parts) >= 2 && parts[0] == "manifest" {
		return parts[1]
	}
	return value
}
