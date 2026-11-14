package behaviour

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

// registerCLISteps binds the steps about a command, its streams, its JSON documents, the environment, the catalog, the
// env file, the output folder, and the secret scan. cur returns the world of the running scenario.
func registerCLISteps(ctx *godog.ScenarioContext, cur func() *world) {
	ctx.Step(`^I run "([^"]*)"$`, func(line string) error {
		argv, err := commandArgs(line)
		if err != nil {
			return err
		}
		return cur().execute(argv)
	})
	ctx.Step(`^the standard output pipe has no reader$`, func() error {
		cur().stdoutClosed = true
		return nil
	})
	ctx.Step(`^the exit status is (\d+)$`, func(want string) error {
		return cur().checkExit(want)
	})
	ctx.Step(`^the standard output is (empty|not empty)$`, func(state string) error {
		return cur().checkEmptiness("stdout", state)
	})
	ctx.Step(`^the standard error contains "([^"]*)"$`, func(text string) error {
		return cur().checkContains("stderr", text, true)
	})
	ctx.Step(`^I keep the standard error as "([^"]*)"$`, func(name string) error {
		return cur().keepStderr(name)
	})
	ctx.Step(`^the standard error is the same as the one kept as "([^"]*)"$`, func(name string) error {
		return cur().checkKeptStderr(name)
	})
	ctx.Step(`^the standard error is (empty|not empty)$`, func(state string) error {
		return cur().checkEmptiness("stderr", state)
	})
	ctx.Step(`^the standard output contains "([^"]*)"$`, func(text string) error {
		return cur().checkContains("stdout", text, true)
	})
	ctx.Step(`^the standard output does not contain "([^"]*)"$`, func(text string) error {
		return cur().checkContains("stdout", text, false)
	})
	ctx.Step(`^the standard error does not contain "([^"]*)"$`, func(text string) error {
		return cur().checkContains("stderr", text, false)
	})
	ctx.Step(`^the standard output is a JSON document with schemaVersion (\d+)$`, func(want string) error {
		return cur().checkSchemaVersion(want)
	})
	ctx.Step(`^the standard output contains no control characters except line breaks$`, func() error {
		return cur().checkNoControl()
	})
	ctx.Step(`^the standard error is a JSON error with the code "([^"]*)" and retryable (true|false)$`,
		func(code, retryable string) error {
			return cur().checkErrorBody(code, retryable, errCodeExact)
		})
	ctx.Step(`^the standard error is a JSON error with a code that starts with "([^"]*)"$`, func(prefix string) error {
		return cur().checkErrorBody(prefix, "", errCodePrefix)
	})
	ctx.Step(`^the standard error is a JSON error with a code other than "([^"]*)"$`, func(code string) error {
		return cur().checkErrorBody(code, "", errCodeOther)
	})
	ctx.Step(`^the JSON field "([^"]*)" of the standard output is (.+)$`, func(path, expected string) error {
		return cur().checkJSONField("stdout", path, expected)
	})
	ctx.Step(`^the JSON field "([^"]*)" of the standard error is (.+)$`, func(path, expected string) error {
		return cur().checkJSONField("stderr", path, expected)
	})
	ctx.Step(`^the value "([^"]*)" never appears in the standard output, the standard error, or the output folder$`,
		func(literal string) error {
			return cur().checkValueAbsent(literal)
		})
	ctx.Step(`^the value of "([^"]*)" never appears in the standard output, the standard error, or the output folder$`,
		func(name string) error {
			value, err := cur().valueOf(name)
			if err != nil {
				return err
			}
			return cur().checkValueAbsent(value)
		})

	ctx.Step(`^the environment variable "([^"]*)" is set to "([^"]*)"$`, func(name, value string) error {
		cur().env[name] = value
		return nil
	})
	ctx.Step(`^the environment variable "([^"]*)" is not set$`, func(name string) error {
		delete(cur().env, name)
		return nil
	})
	ctx.Step(`^the sources catalog holds these sources:$`, func(tbl *godog.Table) error {
		return cur().writeCatalog("sources.yml", tbl)
	})
	ctx.Step(`^the sources catalog file "([^"]*)" holds these sources:$`, func(name string, tbl *godog.Table) error {
		return cur().writeCatalog(name, tbl)
	})
	ctx.Step(`^the registered kinds are the built-in kinds and "([^"]*)"$`, func(name string) error {
		return cur().registerKind(name)
	})
	ctx.Step(`^the sources catalog file "([^"]*)" is absent$`, func(name string) error {
		if cur().be.regularFile(cur().absPath(name)) {
			return fmt.Errorf("the catalog %q must be absent, and it exists", name)
		}
		return nil
	})
	ctx.Step(`^the sources catalog file "([^"]*)" holds this text:$`, func(name string, doc *godog.DocString) error {
		_, err := cur().writeInput(name, doc.Content)
		return err
	})
	ctx.Step(`^the sources catalog file "([^"]*)" is (\d+) KiB of comment lines$`, func(name, kib string) error {
		return cur().writeCommentCatalog(name, kib)
	})
	ctx.Step(`^the sources catalog in the user configuration folder holds these sources:$`, func(tbl *godog.Table) error {
		return cur().writeUserCatalog(tbl)
	})
	ctx.Step(`^the working directory is the folder "([^"]*)"$`, func(rel string) error {
		return cur().setWorkdir(rel)
	})
	ctx.Step(`^the env file "([^"]*)" holds these lines:$`, func(name string, doc *godog.DocString) error {
		return cur().writeEnvFile(name, doc.Content)
	})
	ctx.Step(`^the env file "([^"]*)" in the folder "([^"]*)" holds these lines:$`, func(name, folder string, doc *godog.DocString) error {
		return cur().writeEnvFileIn(folder, name, doc.Content)
	})

	ctx.Step(`^the output folder "([^"]*)" holds exactly (\d+) Markdown files?$`, func(folder, want string) error {
		return cur().checkMarkdownCount(folder, want)
	})
	ctx.Step(`^the output folder "([^"]*)" holds no files$`, func(folder string) error {
		return cur().checkFolderHasNoFiles(folder)
	})
	ctx.Step(`^the output folder "([^"]*)" holds the file "([^"]*)"$`, func(folder, file string) error {
		return cur().checkFolderHoldsFile(folder, file)
	})
	ctx.Step(`^the output folder "([^"]*)" already holds a Markdown file with the text "([^"]*)"$`,
		func(folder, text string) error {
			return cur().seedFolder(folder, "existing.md", text)
		})
	ctx.Step(`^the output folder "([^"]*)" already holds a file named "([^"]*)" with the text "([^"]*)"$`,
		func(folder, file, text string) error {
			return cur().seedFolder(folder, file, text)
		})
	ctx.Step(`^the Markdown file in the output folder "([^"]*)" contains "([^"]*)"$`, func(folder, text string) error {
		return cur().checkMarkdownText(folder, text, matchContains)
	})
	ctx.Step(`^the Markdown file in the output folder "([^"]*)" does not start with "([^"]*)"$`,
		func(folder, text string) error {
			return cur().checkMarkdownText(folder, text, matchNotPrefix)
		})
	ctx.Step(`^the Markdown file in the output folder "([^"]*)" does not contain "([^"]*)"$`,
		func(folder, text string) error {
			return cur().checkMarkdownText(folder, text, matchNotContains)
		})
	ctx.Step(`^the file "([^"]*)" in the output folder "([^"]*)" contains "([^"]*)"$`,
		func(file, folder, text string) error {
			return cur().checkFileText(folder, file, text)
		})
}

// Ways a JSON error code is compared with the expected text.
const (
	errCodeExact = iota
	errCodePrefix
	errCodeOther
)

// Ways a Markdown file's text is compared with the expected text.
const (
	matchContains = iota
	matchNotContains
	matchNotPrefix
)

// streamText returns the text of one stream of the last command.
func (w *world) streamText(name string) (string, error) {
	r, err := w.lastRun()
	if err != nil {
		return "", err
	}
	switch name {
	case "stdout":
		return r.stdout, nil
	case "stderr":
		return r.stderr, nil
	}
	return "", fmt.Errorf("there is no stream %q", name)
}

// checkExit checks the exit status of the last command.
func (w *world) checkExit(want string) error {
	r, err := w.lastRun()
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(want)
	if err != nil {
		return err
	}
	if r.exit != n {
		return fmt.Errorf("the exit status is %d, want %d (stdout %q, stderr %q)", r.exit, n, r.stdout, r.stderr)
	}
	return nil
}

// checkEmptiness checks that a stream holds no bytes, or that it holds some.
func (w *world) checkEmptiness(stream, state string) error {
	text, err := w.streamText(stream)
	if err != nil {
		return err
	}
	empty := text == ""
	if state == "empty" && !empty {
		return fmt.Errorf("the %s is not empty: %q", stream, text)
	}
	if state == "not empty" && empty {
		return fmt.Errorf("the %s is empty", stream)
	}
	return nil
}

// checkContains checks that a stream holds the text, or that it does not.
func (w *world) checkContains(stream, text string, want bool) error {
	got, err := w.streamText(stream)
	if err != nil {
		return err
	}
	if strings.Contains(got, text) != want {
		if want {
			return fmt.Errorf("the %s does not contain %q: %q", stream, text, got)
		}
		return fmt.Errorf("the %s contains %q: %q", stream, text, got)
	}
	return nil
}

// keepStderr stores the standard error of the last command under a name, so a later command can be compared with it.
func (w *world) keepStderr(name string) error {
	text, err := w.streamText("stderr")
	if err != nil {
		return err
	}
	if w.kept == nil {
		w.kept = map[string]string{}
	}
	w.kept[name] = text
	return nil
}

// checkKeptStderr checks that the standard error of the last command is the text kept under the name. An empty text is
// refused, so the check is never true because nothing was written.
func (w *world) checkKeptStderr(name string) error {
	want, ok := w.kept[name]
	if !ok {
		return fmt.Errorf("no standard error was kept as %q", name)
	}
	got, err := w.streamText("stderr")
	if err != nil {
		return err
	}
	if got == "" || want == "" {
		return errors.New("an empty standard error cannot be compared")
	}
	if got != want {
		return fmt.Errorf("the standard error is %q, want the kept text %q", got, want)
	}
	return nil
}

// checkSchemaVersion checks the schemaVersion of the JSON document on stdout.
func (w *world) checkSchemaVersion(want string) error {
	return w.checkJSONField("stdout", "schemaVersion", want)
}

// checkNoControl checks that stdout holds no control byte except a line break or a tab.
func (w *world) checkNoControl() error {
	text, err := w.streamText("stdout")
	if err != nil {
		return err
	}
	if hasControlBytes(text) {
		return errors.New("the standard output holds a control character")
	}
	return nil
}

// errorBody decodes the failure body of stderr, the object under the error key.
func (w *world) errorBody() (map[string]any, error) {
	text, err := w.streamText("stderr")
	if err != nil {
		return nil, err
	}
	doc, err := parseJSON(text)
	if err != nil {
		return nil, fmt.Errorf("the standard error is not a JSON document: %w", err)
	}
	root, _ := doc.(map[string]any)
	obj, ok := root["error"].(map[string]any)
	if !ok {
		return nil, errors.New("the standard error has no error object")
	}
	return obj, nil
}

// checkErrorBody checks the code and the retryable flag of the failure body on stderr. The mode says how the code is
// compared: exactly, by prefix, or as anything other than the code.
func (w *world) checkErrorBody(code, retryable string, mode int) error {
	obj, err := w.errorBody()
	if err != nil {
		return err
	}
	got, _ := obj["code"].(string)
	switch mode {
	case errCodeExact:
		if got != code {
			return fmt.Errorf("the error code is %q, want %q", got, code)
		}
		want := retryable == "true"
		if r, _ := obj["retryable"].(bool); r != want {
			return fmt.Errorf("the error is retryable %v, want %v", r, want)
		}
	case errCodePrefix:
		if !strings.HasPrefix(got, code) {
			return fmt.Errorf("the error code %q does not start with %q", got, code)
		}
	case errCodeOther:
		if got == code {
			return fmt.Errorf("the error code is %q, and it must differ", got)
		}
	}
	return nil
}

// checkJSONField checks one field of the JSON document on a stream against the expected text.
func (w *world) checkJSONField(stream, path, expected string) error {
	text, err := w.streamText(stream)
	if err != nil {
		return err
	}
	doc, err := parseJSON(text)
	if err != nil {
		return fmt.Errorf("the %s is not a JSON document: %w", stream, err)
	}
	got, found := lookupPath(doc, path)
	if !valueMatches(got, found, expected) {
		return fmt.Errorf("the field %q of the %s is %v (found %v), want %s", path, stream, got, found, expected)
	}
	return nil
}

// checkValueAbsent checks that a value is in neither stream nor any file the command wrote. The inputs the scenario wrote
// itself are skipped, because they hold the value by design. The value is the text itself, never a variable name.
func (w *world) checkValueAbsent(value string) error {
	if value == "" {
		return errors.New("an empty value cannot be checked for absence")
	}
	r, err := w.lastRun()
	if err != nil {
		return err
	}
	if strings.Contains(r.stdout, value) {
		return errors.New("the value appears in the standard output")
	}
	if strings.Contains(r.stderr, value) {
		return errors.New("the value appears in the standard error")
	}
	files, err := w.scanFiles()
	if err != nil {
		return err
	}
	for path, content := range files {
		if strings.Contains(content, value) {
			return fmt.Errorf("the value appears in the file %s", path)
		}
	}
	return nil
}

// writeCatalog writes a sources catalog from a table of sources, in the YAML form the catalog loader reads.
func (w *world) writeCatalog(name string, tbl *godog.Table) error {
	rows, err := tableRows(tbl)
	if err != nil {
		return err
	}
	_, err = w.writeInput(name, renderCatalog(rows))
	return err
}

// renderCatalog renders the rows of a sources table as a YAML catalog. A blank cell is left out of its source.
func renderCatalog(rows []map[string]string) string {
	var b strings.Builder
	b.WriteString("version: 1\nsources:\n")
	for _, r := range rows {
		first := true
		line := func(key, value string) {
			if value == "" {
				return
			}
			prefix := "    "
			if first {
				prefix = "  - "
				first = false
			}
			fmt.Fprintf(&b, "%s%s: %s\n", prefix, key, value)
		}
		line("id", r["id"])
		line("kind", r["kind"])
		line("tags", "["+tagList(r["tags"])+"]")
		line("access", r["access"])
		line("endpoint", r["endpoint"])
		line("credential_env", r["credential_env"])
		line("identity_env", r["identity_env"])
	}
	return b.String()
}

// tagList turns a comma-separated tag cell into the items of a YAML flow list.
func tagList(cell string) string {
	var tags []string
	for _, t := range strings.Split(cell, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return strings.Join(tags, ", ")
}

// writeEnvFile writes an env file into the scenario folder. It sets no variable: the file is found by the rule for the
// working directory, so a scenario that uses it proves that rule. The values it holds are remembered, so the secret scan
// can look for them.
func (w *world) writeEnvFile(name, content string) error {
	return w.writeEnvFileIn("", name, content)
}

// writeEnvFileIn writes an env file into a folder of the scenario, creating the folder first, and remembers its values.
func (w *world) writeEnvFileIn(folder, name, content string) error {
	if folder != "" {
		if err := w.be.mkdir(w.absPath(folder)); err != nil {
			return err
		}
	}
	if _, err := w.writeInput(path.Join(folder, name), content); err != nil {
		return err
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			w.envFileVars[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return nil
}

// setWorkdir makes a folder of the scenario the working directory of the commands that follow.
func (w *world) setWorkdir(rel string) error {
	if err := w.be.mkdir(w.absPath(rel)); err != nil {
		return err
	}
	w.cwd = rel
	return nil
}

// writeUserCatalog writes a sources catalog into the user configuration folder, which the scenario points at through
// XDG_CONFIG_HOME.
func (w *world) writeUserCatalog(tbl *godog.Table) error {
	rows, err := tableRows(tbl)
	if err != nil {
		return err
	}
	if err := w.be.mkdir(w.absPath("config/retriever")); err != nil {
		return err
	}
	_, err = w.writeInput("config/retriever/sources.yml", renderCatalog(rows))
	return err
}

// writeCommentCatalog writes a catalog that is only comment lines, padded to the size in KiB. It is valid YAML of the
// wrong size, so the size rule is what refuses it.
func (w *world) writeCommentCatalog(name, kib string) error {
	n, err := strconv.Atoi(kib)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("version: 1\nsources: []\n")
	line := "# " + strings.Repeat("x", 61) + "\n"
	for b.Len() < n*1024 {
		b.WriteString(line)
	}
	_, err = w.writeInput(name, b.String())
	return err
}

// checkMarkdownCount checks how many Markdown files an output folder holds directly.
func (w *world) checkMarkdownCount(folder, want string) error {
	n, err := strconv.Atoi(want)
	if err != nil {
		return err
	}
	files, err := w.folderFiles(folder)
	if err != nil {
		return err
	}
	count := 0
	for name := range files {
		if strings.HasSuffix(name, ".md") {
			count++
		}
	}
	if count != n {
		return fmt.Errorf("the output folder %q holds %d Markdown files, want %d", folder, count, n)
	}
	return nil
}

// checkFolderHasNoFiles checks that an output folder holds no file.
func (w *world) checkFolderHasNoFiles(folder string) error {
	files, err := w.folderFiles(folder)
	if err != nil {
		return err
	}
	if len(files) != 0 {
		return fmt.Errorf("the output folder %q holds %d files", folder, len(files))
	}
	return nil
}

// checkFolderHoldsFile checks that an output folder holds the named file.
func (w *world) checkFolderHoldsFile(folder, file string) error {
	files, err := w.folderFiles(folder)
	if err != nil {
		return err
	}
	if _, ok := files[file]; !ok {
		return fmt.Errorf("the output folder %q does not hold the file %q", folder, file)
	}
	return nil
}

// seedFolder writes a file into an output folder before the command runs. The file is an input, so the secret scan skips
// it.
func (w *world) seedFolder(folder, file, text string) error {
	if _, err := w.ensureOutputFolder(folder); err != nil {
		return err
	}
	_, err := w.writeInput(filepath.ToSlash(filepath.Join(folder, file)), text)
	return err
}

// checkMarkdownText checks the Markdown files of an output folder against one text. The mode says how: the text must
// appear in some file, must not appear in any file, or must not be the start of any file.
func (w *world) checkMarkdownText(folder, text string, mode int) error {
	files, err := w.folderFiles(folder)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasSuffix(name, ".md") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	switch mode {
	case matchContains:
		if len(names) == 0 {
			return fmt.Errorf("the output folder %q holds no Markdown file", folder)
		}
		for _, name := range names {
			if strings.Contains(files[name], text) {
				return nil
			}
		}
		return fmt.Errorf("no Markdown file in %q contains %q", folder, text)
	case matchNotContains:
		for _, name := range names {
			if strings.Contains(files[name], text) {
				return fmt.Errorf("the Markdown file %s contains %q", name, text)
			}
		}
	case matchNotPrefix:
		for _, name := range names {
			if strings.HasPrefix(files[name], text) {
				return fmt.Errorf("the Markdown file %s starts with %q", name, text)
			}
		}
	}
	return nil
}

// checkFileText checks that a named file in an output folder contains the text.
func (w *world) checkFileText(folder, file, text string) error {
	files, err := w.folderFiles(folder)
	if err != nil {
		return err
	}
	content, ok := files[file]
	if !ok {
		return fmt.Errorf("the output folder %q does not hold the file %q", folder, file)
	}
	if !strings.Contains(content, text) {
		return fmt.Errorf("the file %q does not contain %q", file, text)
	}
	return nil
}

// tableRows turns a Gherkin table into one map per row, keyed by the header cell. Every row must have a cell for each
// header.
func tableRows(tbl *godog.Table) ([]map[string]string, error) {
	if tbl == nil || len(tbl.Rows) < 2 {
		return nil, errors.New("the table needs a header row and at least one row")
	}
	var header []string
	for _, c := range tbl.Rows[0].Cells {
		header = append(header, strings.TrimSpace(c.Value))
	}
	var rows []map[string]string
	for _, row := range tbl.Rows[1:] {
		if len(row.Cells) != len(header) {
			return nil, fmt.Errorf("a table row has %d cells, the header has %d", len(row.Cells), len(header))
		}
		m := make(map[string]string, len(header))
		for i, c := range row.Cells {
			m[header[i]] = strings.TrimSpace(c.Value)
		}
		rows = append(rows, m)
	}
	return rows, nil
}
