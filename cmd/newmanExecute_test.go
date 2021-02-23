package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	sliceUtils "github.com/SAP/jenkins-library/pkg/piperutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type executedScript struct {
	shell  string
	script string
	envs   []string
}

type newmanExecuteMockUtils struct {
	// *mock.ExecMockRunner
	// *mock.FilesMock
	errorOnGlob            bool
	errorOnNewmanInstall   bool
	errorOnRunShell        bool
	errorOnNewmanExecution bool
	errorOnLoggingNode     bool
	errorOnLoggingNpm      bool
	executedScripts        []executedScript
	filesToFind            []string
	commandIndex           int
}

func newNewmanExecuteMockUtils() newmanExecuteMockUtils {
	return newmanExecuteMockUtils{
		filesToFind: []string{"localFile.json", "localFile2.json"},
	}
}

func TestRunNewmanExecute(t *testing.T) {
	t.Parallel()

	allFineConfig := newmanExecuteOptions{
		NewmanCollection:     "**.json",
		NewmanEnvironment:    "env.json",
		NewmanGlobals:        "globals.json",
		NewmanInstallCommand: "npm install newman --global --quiet",
		NewmanRunCommand:     "run {{.NewmanCollection}} --environment {{.Config.NewmanEnvironment}} --globals {{.Config.NewmanGlobals}} --reporters junit,html --reporter-junit-export target/newman/TEST-{{.CollectionDisplayName}}.xml --reporter-html-export target/newman/TEST-{{.CollectionDisplayName}}.html",
	}

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.NoError(t, err)
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "node --version", envs: []string{"NPM_CONFIG_PREFIX=~/.npm-global"}})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "npm --version"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "npm install newman --global --quiet"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "/home/node/.npm-global/bin/newman run localFile.json --environment env.json --globals globals.json --reporters junit,html --reporter-junit-export target/newman/TEST-localFile.xml --reporter-html-export target/newman/TEST-localFile.html --suppress-exit-code"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "/home/node/.npm-global/bin/newman run localFile2.json --environment env.json --globals globals.json --reporters junit,html --reporter-junit-export target/newman/TEST-localFile2.xml --reporter-html-export target/newman/TEST-localFile2.html --suppress-exit-code"})
	})

	t.Run("happy path with fail on error", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		fineConfig := allFineConfig
		fineConfig.FailOnError = true

		// test
		err := runNewmanExecute(&fineConfig, &utils)

		// assert
		assert.NoError(t, err)
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "node --version", envs: []string{"NPM_CONFIG_PREFIX=~/.npm-global"}})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "npm --version"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "npm install newman --global --quiet"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "/home/node/.npm-global/bin/newman run localFile.json --environment env.json --globals globals.json --reporters junit,html --reporter-junit-export target/newman/TEST-localFile.xml --reporter-html-export target/newman/TEST-localFile.html"})
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "/home/node/.npm-global/bin/newman run localFile2.json --environment env.json --globals globals.json --reporters junit,html --reporter-junit-export target/newman/TEST-localFile2.xml --reporter-html-export target/newman/TEST-localFile2.html"})
	})

	t.Run("error on newman execution", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		utils.errorOnNewmanExecution = true

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.EqualError(t, err, "The execution of the newman tests failed, see the log for details.: error on newman execution")
	})

	t.Run("error on newman installation", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		utils.errorOnNewmanInstall = true

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.EqualError(t, err, "error installing newman: error on newman install")
	})

	t.Run("error on npm version logging", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		utils.errorOnLoggingNpm = true

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.EqualError(t, err, "error logging npm version: error on RunShell")
	})

	t.Run("error on template resolution", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		config := allFineConfig
		config.NewmanRunCommand = "this is my erroneous command {{.collectionDisplayName}"

		// test
		err := runNewmanExecute(&config, &utils)

		// assert
		assert.EqualError(t, err, "could not parse newman command template: template: template:1: unexpected \"}\" in operand")
	})

	t.Run("error on file search", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		utils.filesToFind = nil

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.EqualError(t, err, "no collection found with pattern '**.json'")
	})

	t.Run("no newman file", func(t *testing.T) {
		t.Parallel()
		// init

		utils := newNewmanExecuteMockUtils()
		utils.errorOnGlob = true

		// test
		err := runNewmanExecute(&allFineConfig, &utils)

		// assert
		assert.EqualError(t, err, "Could not execute global search for '**.json': error on Glob")
	})
}

func TestDefineCollectionDisplayName(t *testing.T) {
	t.Parallel()

	t.Run("normal path", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join("dir1", "dir2", "fancyFile.txt")
		result := defineCollectionDisplayName(path)
		assert.Equal(t, "dir1_dir2_fancyFile", result)
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join("dir1", "dir2", "dir3")
		result := defineCollectionDisplayName(path)
		assert.Equal(t, "dir1_dir2_dir3", result)
	})

	t.Run("directory with dot prefix", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(".dir1", "dir2", "dir3", "file.json")
		result := defineCollectionDisplayName(path)
		assert.Equal(t, "dir1_dir2_dir3_file", result)
	})

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(".")
		result := defineCollectionDisplayName(path)
		assert.Equal(t, "", result)
	})
}

func TestResolveTemplate(t *testing.T) {
	t.Parallel()

	t.Run("nothing to replace", func(t *testing.T) {
		t.Parallel()

		config := newmanExecuteOptions{NewmanRunCommand: "this is my fancy command"}

		cmd, err := resolveTemplate(&config, "collectionsDisplayName")
		assert.NoError(t, err)
		assert.Equal(t, "this is my fancy command", cmd)
	})

	t.Run("replace display name", func(t *testing.T) {
		t.Parallel()

		config := newmanExecuteOptions{NewmanRunCommand: "this is my fancy command {{.CollectionDisplayName}}"}

		cmd, err := resolveTemplate(&config, "theDisplayName")
		assert.NoError(t, err)
		assert.Equal(t, "this is my fancy command theDisplayName", cmd)
	})

	t.Run("replace config Verbose", func(t *testing.T) {
		t.Parallel()

		config := newmanExecuteOptions{
			NewmanRunCommand: "this is my fancy command {{.Config.Verbose}}",
			Verbose:          false,
		}

		cmd, err := resolveTemplate(&config, "theDisplayName")
		assert.NoError(t, err)
		assert.Equal(t, "this is my fancy command false", cmd)
	})

	t.Run("error when parameter cannot be resolved", func(t *testing.T) {
		t.Parallel()

		config := newmanExecuteOptions{NewmanRunCommand: "this is my fancy command {{.collectionDisplayName}}"}

		_, err := resolveTemplate(&config, "theDisplayName")
		assert.EqualError(t, err, "error on executing template: template: template:1:27: executing \"template\" at <.collectionDisplayName>: can't evaluate field collectionDisplayName in type cmd.TemplateConfig")
	})

	t.Run("error when template cannot be parsed", func(t *testing.T) {
		t.Parallel()

		config := newmanExecuteOptions{NewmanRunCommand: "this is my fancy command {{.collectionDisplayName}"}

		_, err := resolveTemplate(&config, "theDisplayName")
		assert.EqualError(t, err, "could not parse newman command template: template: template:1: unexpected \"}\" in operand")
	})
}

func TestLogVersions(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		utils := newNewmanExecuteMockUtils()

		err := logVersions(&utils)
		assert.NoError(t, err)
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "npm --version"})
	})

	t.Run("error in node execution", func(t *testing.T) {
		utils := newNewmanExecuteMockUtils()
		utils.errorOnLoggingNode = true

		err := logVersions(&utils)
		assert.EqualError(t, err, "error logging node version: error on RunShell")
	})

	t.Run("error in npm execution", func(t *testing.T) {
		utils := newNewmanExecuteMockUtils()
		utils.errorOnLoggingNpm = true

		err := logVersions(&utils)
		assert.EqualError(t, err, "error logging npm version: error on RunShell")
		assert.Contains(t, utils.executedScripts, executedScript{shell: "/bin/sh", script: "node --version"})
	})
}

func (e *newmanExecuteMockUtils) Glob(string) (matches []string, err error) {
	if e.errorOnGlob {
		return nil, errors.New("error on Glob")
	}

	return e.filesToFind, nil
}

func (e *newmanExecuteMockUtils) RunShell(shell string, script string) error {
	params := strings.Split(script, " ")
	executable := params[0]
	if e.errorOnRunShell {
		return errors.New("error on RunShell")
	}
	if e.errorOnLoggingNode && executable == "node" && params[1] == "--version" {
		return errors.New("error on RunShell")
	}
	if e.errorOnLoggingNpm && executable == "npm" && params[1] == "--version" {
		return errors.New("error on RunShell")
	}
	if e.errorOnNewmanExecution && strings.Contains(executable, "newman") {
		return errors.New("error on newman execution")
	}
	if e.errorOnNewmanInstall && sliceUtils.ContainsString(params, "install") {
		return errors.New("error on newman install")
	}

	length := len(e.executedScripts)
	if length < e.commandIndex+1 {
		e.executedScripts = append(e.executedScripts, executedScript{})
		length++
	}

	e.executedScripts[length-1].shell = shell
	e.executedScripts[length-1].script = script
	e.commandIndex++

	return nil
}

func (e *newmanExecuteMockUtils) SetEnv(env []string) {
	// set envs to current script command in mock (only)
	length := len(e.executedScripts)
	if length < e.commandIndex+1 {
		e.executedScripts = append(e.executedScripts, executedScript{})
		length++
	}
	e.executedScripts[length-1].envs = append(e.executedScripts[length-1].envs, env...)
}