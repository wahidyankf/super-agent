package behaviour

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// registerProcessSteps binds the steps that start the long-running protocol servers (serve and mcp) and check their
// answers. These steps run at the end-to-end level, where the built binary is available. cur returns the running
// scenario's world.
func registerProcessSteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^retriever serve is running on the loopback address$`, func() error {
		return cur().startServe()
	})
	ctx.Step(`^retriever serve receives this GraphQL request:$`, func(doc *godog.DocString) error {
		return cur().postGraphQL(doc.Content)
	})
	ctx.Step(`^the HTTP status is (\d+)$`, func(want string) error {
		return cur().checkHTTPStatus(want)
	})
	ctx.Step(`^the GraphQL response field "([^"]*)" is (.+)$`, func(path, expected string) error {
		return cur().checkGraphQLField(path, expected)
	})
	ctx.Step(`^the GraphQL response has no errors$`, func() error {
		return cur().checkGraphQLErrors(false)
	})
	ctx.Step(`^the GraphQL response has an error$`, func() error {
		return cur().checkGraphQLErrors(true)
	})
	ctx.Step(`^the GraphQL response does not contain "([^"]*)"$`, func(text string) error {
		return cur().checkGraphQLText(text)
	})
	ctx.Step(`^the server listens on the loopback address only$`, func() error {
		w := cur()
		if err := w.startServe(); err != nil {
			return err
		}
		if !isLoopback(w.serve.addr) {
			return fmt.Errorf("the server listens on %s, which is not a loopback address", w.serve.addr)
		}
		return nil
	})

	ctx.Step(`^an MCP client starts "([^"]*)"$`, func(line string) error {
		return cur().startMCP(line)
	})
	ctx.Step(`^the client lists the tools$`, func() error {
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		return m.listTools()
	})
	ctx.Step(`^the tool list has the tool "([^"]*)"$`, func(name string) error {
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		if _, ok := m.toolNamed(name); !ok {
			return fmt.Errorf("the tool list has no tool %q", name)
		}
		return nil
	})
	ctx.Step(`^the tool list has no tool named "([^"]*)"$`, func(name string) error {
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		if _, ok := m.toolNamed(name); ok {
			return fmt.Errorf("the tool list has the tool %q", name)
		}
		return nil
	})
	ctx.Step(`^the tool "([^"]*)" is marked read-only$`, func(name string) error {
		return cur().checkToolHint(name, func(t mcpTool) bool { return t.readOnly }, "read-only")
	})
	ctx.Step(`^the tool "([^"]*)" is marked destructive$`, func(name string) error {
		return cur().checkToolHint(name, func(t mcpTool) bool { return t.destructive }, "destructive")
	})
	ctx.Step(`^the client calls the tool "([^"]*)" with these arguments:$`, func(name string, doc *godog.DocString) error {
		var args map[string]any
		if err := json.Unmarshal([]byte(doc.Content), &args); err != nil {
			return fmt.Errorf("the tool arguments are not a JSON object: %w", err)
		}
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		return m.callTool(name, args)
	})
	ctx.Step(`^the client calls the tool "([^"]*)" with no arguments$`, func(name string) error {
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		return m.callTool(name, nil)
	})
	ctx.Step(`^the tool result is an error$`, func() error {
		return cur().checkToolResultError(true)
	})
	ctx.Step(`^the tool result is not an error$`, func() error {
		return cur().checkToolResultError(false)
	})
	ctx.Step(`^the tool result text contains "([^"]*)"$`, func(text string) error {
		return cur().checkToolText(text, true)
	})
	ctx.Step(`^the tool result text does not contain "([^"]*)"$`, func(text string) error {
		return cur().checkToolText(text, false)
	})
	ctx.Step(`^the tool result text contains no control characters except line breaks$`, func() error {
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		if m.call == nil {
			return errors.New("no tool was called")
		}
		if hasControlBytes(m.call.text) {
			return errors.New("the tool result text holds a control character")
		}
		return nil
	})
	ctx.Step(`^the whole tool result is at most (\d+) bytes$`, func(max string) error {
		n, err := strconv.Atoi(max)
		if err != nil {
			return err
		}
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		if m.call == nil {
			return errors.New("no tool was called")
		}
		if m.call.size > n {
			return fmt.Errorf("the tool result is %d bytes, want at most %d", m.call.size, n)
		}
		return nil
	})
	ctx.Step(`^the MCP server exits with status (\d+)$`, func(want string) error {
		n, err := strconv.Atoi(want)
		if err != nil {
			return err
		}
		m, err := cur().mcpClient()
		if err != nil {
			return err
		}
		if err := m.finish(); err != nil {
			return err
		}
		if m.exit != n {
			return fmt.Errorf("the MCP server exited with status %d, want %d", m.exit, n)
		}
		return nil
	})
}

// mcpClient returns the MCP session of the scenario, or an error when no client has started.
func (w *world) mcpClient() (*mcpSession, error) {
	if w.mcp == nil {
		return nil, errors.New("no MCP client has started in this scenario")
	}
	return w.mcp, nil
}

// toolNamed returns the tool with the name, from the last tool list.
func (m *mcpSession) toolNamed(name string) (mcpTool, bool) {
	for _, t := range m.tools {
		if t.name == name {
			return t, true
		}
	}
	return mcpTool{}, false
}

// checkToolHint checks one hint of a listed tool.
func (w *world) checkToolHint(name string, hint func(mcpTool) bool, label string) error {
	m, err := w.mcpClient()
	if err != nil {
		return err
	}
	t, ok := m.toolNamed(name)
	if !ok {
		return fmt.Errorf("the tool list has no tool %q", name)
	}
	if !hint(t) {
		return fmt.Errorf("the tool %q is not marked %s", name, label)
	}
	return nil
}

// checkToolResultError checks whether the last tool result is an error.
func (w *world) checkToolResultError(want bool) error {
	m, err := w.mcpClient()
	if err != nil {
		return err
	}
	if m.call == nil {
		return errors.New("no tool was called")
	}
	if m.call.isError != want {
		return fmt.Errorf("the tool result is an error %v, want %v", m.call.isError, want)
	}
	return nil
}

// checkToolText checks whether the text of the last tool result contains a text, or does not.
func (w *world) checkToolText(text string, want bool) error {
	m, err := w.mcpClient()
	if err != nil {
		return err
	}
	if m.call == nil {
		return errors.New("no tool was called")
	}
	if strings.Contains(m.call.text, text) != want {
		return fmt.Errorf("the tool result text contains %q: %v, want %v", text, !want, want)
	}
	return nil
}

// checkHTTPStatus checks the status of the last GraphQL answer.
func (w *world) checkHTTPStatus(want string) error {
	if w.reply == nil {
		return errors.New("no GraphQL request was sent in this scenario")
	}
	n, err := strconv.Atoi(want)
	if err != nil {
		return err
	}
	if w.reply.status != n {
		return fmt.Errorf("the HTTP status is %d, want %d (body %q)", w.reply.status, n, w.reply.body)
	}
	return nil
}

// checkGraphQLField checks one field of the last GraphQL answer against the expected text.
func (w *world) checkGraphQLField(path, expected string) error {
	if w.reply == nil {
		return errors.New("no GraphQL request was sent in this scenario")
	}
	got, found := lookupPath(w.reply.doc, path)
	if !valueMatches(got, found, expected) {
		return fmt.Errorf("the GraphQL field %q is %v (found %v), want %s", path, got, found, expected)
	}
	return nil
}

// checkGraphQLErrors checks whether the last GraphQL answer carries errors, or none.
func (w *world) checkGraphQLErrors(wantErrors bool) error {
	if w.reply == nil {
		return errors.New("no GraphQL request was sent in this scenario")
	}
	root, _ := w.reply.doc.(map[string]any)
	list, _ := root["errors"].([]any)
	if (len(list) > 0) != wantErrors {
		return fmt.Errorf("the GraphQL response has %d errors, want an error %v", len(list), wantErrors)
	}
	return nil
}

// checkGraphQLText checks that the raw body of the last GraphQL answer does not contain a text.
func (w *world) checkGraphQLText(text string) error {
	if w.reply == nil {
		return errors.New("no GraphQL request was sent in this scenario")
	}
	if strings.Contains(w.reply.body, text) {
		return fmt.Errorf("the GraphQL response contains %q", text)
	}
	return nil
}
