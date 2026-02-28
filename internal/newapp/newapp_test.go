package newapp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSingleRepoSuccess(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputBaseDir := filepath.Join(t.TempDir(), "single")
	err := Run("github.com/acme/order", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputBaseDir,
		SkipPostTasks: true,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	outputDir := filepath.Join(outputBaseDir, "order-api")
	if !isDir(outputDir) {
		t.Fatalf("single repo mode should create app dir: %s", outputDir)
	}

	if pathExists(filepath.Join(outputDir, "app")) {
		t.Fatalf("single repo mode should remove app directory")
	}

	goMain := mustReadFile(t, filepath.Join(outputDir, "cmd", "main.go"))
	if strings.Contains(goMain, "app_layout") {
		t.Fatalf("single repo go file still contains app_layout: %s", goMain)
	}
	if !strings.Contains(goMain, "github.com/acme/order/internal/conf") {
		t.Fatalf("single repo go import not rewritten correctly: %s", goMain)
	}
	if strings.Contains(goMain, "/app/order-api/") {
		t.Fatalf("single repo go import still contains /app/order-api: %s", goMain)
	}

	protoConf := mustReadFile(t, filepath.Join(outputDir, "internal", "conf", "conf.proto"))
	if !strings.Contains(protoConf, "app.order_api") {
		t.Fatalf("proto package was not rewritten to underscore name: %s", protoConf)
	}
	if strings.Contains(protoConf, "app_layout") {
		t.Fatalf("proto file still contains app_layout: %s", protoConf)
	}

	goMod := mustReadFile(t, filepath.Join(outputDir, "go.mod"))
	if !strings.Contains(goMod, "github.com/acme/order") {
		t.Fatalf("go.mod module path not rewritten: %s", goMod)
	}

	golangCILint := mustReadFile(t, filepath.Join(outputDir, ".golangci.yml"))
	if strings.Contains(golangCILint, templateModule) {
		t.Fatalf(".golangci.yml still contains template module path: %s", golangCILint)
	}
	if !strings.Contains(golangCILint, "github.com/acme/order") {
		t.Fatalf(".golangci.yml module path not rewritten: %s", golangCILint)
	}

	if pathExists(filepath.Join(outputDir, ".git")) {
		t.Fatalf("copied output should not contain source .git directory")
	}
}

func TestRunSingleRepoOutputBaseNotEmpty(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputBaseDir := t.TempDir()
	writeTextFile(t, filepath.Join(outputBaseDir, "exists.txt"), "already here")

	err := Run("github.com/acme/order", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputBaseDir,
		SkipPostTasks: true,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	outputDir := filepath.Join(outputBaseDir, "order-api")
	if !isDir(outputDir) {
		t.Fatalf("single repo mode should create app dir under output base: %s", outputDir)
	}
}

func TestRunSingleRepoTargetExistsAndNotEmpty(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputBaseDir := t.TempDir()
	writeTextFile(t, filepath.Join(outputBaseDir, "order-api", "exists.txt"), "already here")

	err := Run("github.com/acme/order", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputBaseDir,
		SkipPostTasks: true,
	})
	if err == nil {
		t.Fatalf("expected error when target app directory is not empty")
	}
	if !strings.Contains(err.Error(), "单仓模式要求输出目录为空") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMonoRepoWithModulePathSuccess(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputDir := t.TempDir()

	err := Run("github.com/acme/mono", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputDir,
		MonoRepo:      true,
		SkipPostTasks: true,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	target := filepath.Join(outputDir, "order-api")
	if !isDir(target) {
		t.Fatalf("mono repo mode should create %s", target)
	}

	if !isDir(filepath.Join(target, "app", "order-api")) {
		t.Fatalf("mono-with-module mode should rename app/app_layout to app/order-api")
	}

	goMain := mustReadFile(t, filepath.Join(target, "app", "order-api", "cmd", "main.go"))
	if !strings.Contains(goMain, "github.com/acme/mono/app/order-api/internal/conf") {
		t.Fatalf("mono repo go import not rewritten correctly: %s", goMain)
	}

	protoConf := mustReadFile(t, filepath.Join(target, "app", "order-api", "internal", "conf", "conf.proto"))
	if !strings.Contains(protoConf, "app.order_api") {
		t.Fatalf("proto package was not rewritten in mono mode: %s", protoConf)
	}
}

func TestRunMonoRepoOnlyAppNameSuccess(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputDir := t.TempDir()
	writeTextFile(t, filepath.Join(outputDir, "go.mod"), "module github.com/acme/mono\n")
	mustMkdirAll(t, filepath.Join(outputDir, "app"))

	err := Run("", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputDir,
		MonoRepo:      true,
		SkipPostTasks: true,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	target := filepath.Join(outputDir, "app", "order-api")
	if !isDir(target) {
		t.Fatalf("mono-only-app mode should create %s", target)
	}

	goMain := mustReadFile(t, filepath.Join(target, "cmd", "main.go"))
	if !strings.Contains(goMain, "github.com/acme/mono/app/order-api/internal/conf") {
		t.Fatalf("mono-only-app go import not rewritten correctly: %s", goMain)
	}
}

func TestRunMonoRepoMissingAppDir(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputDir := t.TempDir()
	writeTextFile(t, filepath.Join(outputDir, "go.mod"), "module github.com/acme/mono\n")

	err := Run("", "order-api", Options{
		LayoutSource:  layoutDir,
		OutputDir:     outputDir,
		MonoRepo:      true,
		SkipPostTasks: true,
	})
	if err == nil {
		t.Fatalf("expected error when mono mode has no app dir")
	}
	if !strings.Contains(err.Error(), "大仓模式(仅应用名)要求输出目录下已存在 app 子目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchLayoutLocalSourceAndCleanup(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	layoutRoot, cleanup, err := fetchLayout(layoutDir)
	if err != nil {
		t.Fatalf("fetchLayout() returned error: %v", err)
	}
	if !isDir(layoutRoot) {
		t.Fatalf("layout root was not created: %s", layoutRoot)
	}

	if !pathExists(filepath.Join(layoutRoot, "app", "app_layout", "cmd", "main.go")) {
		t.Fatalf("layout source was not copied")
	}

	cleanup()
	if pathExists(layoutRoot) {
		t.Fatalf("cleanup should remove temporary layout root")
	}
}

func TestReplaceInFileSkipsBinary(t *testing.T) {
	t.Parallel()

	binFile := filepath.Join(t.TempDir(), "raw.bin")
	original := []byte{0x00, 'a', 'p', 'p', '_', 'l', 'a', 'y', 'o', 'u', 't'}
	if err := os.WriteFile(binFile, original, 0o644); err != nil {
		t.Fatalf("write binary file failed: %v", err)
	}

	if err := replaceInFile(binFile, "app_layout", "order-api"); err != nil {
		t.Fatalf("replaceInFile() returned error: %v", err)
	}

	after, err := os.ReadFile(binFile)
	if err != nil {
		t.Fatalf("read binary file failed: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("binary file should not be modified")
	}
}

func TestCopyDirSkipsGitAndCopiesSymlink(t *testing.T) {
	t.Parallel()

	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	writeTextFile(t, filepath.Join(src, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeTextFile(t, filepath.Join(src, "dir", "a.txt"), "hello")
	writeTextFile(t, filepath.Join(src, "dir", ".git"), "gitdir: ../../.git/modules/dir\n")
	if err := os.Symlink("a.txt", filepath.Join(src, "dir", "link.txt")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir() returned error: %v", err)
	}

	if pathExists(filepath.Join(dst, ".git")) {
		t.Fatalf(".git directory should be skipped when copying")
	}
	if pathExists(filepath.Join(dst, "dir", ".git")) {
		t.Fatalf(".git file should be skipped when copying")
	}

	content := mustReadFile(t, filepath.Join(dst, "dir", "a.txt"))
	if content != "hello" {
		t.Fatalf("unexpected copied content: %s", content)
	}

	linkPath := filepath.Join(dst, "dir", "link.txt")
	linkTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("read copied symlink failed: %v", err)
	}
	if linkTarget != "a.txt" {
		t.Fatalf("unexpected symlink target: %s", linkTarget)
	}
}

func TestMoveDirContentsConflict(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")
	mustMkdirAll(t, src)
	mustMkdirAll(t, dst)
	writeTextFile(t, filepath.Join(src, "x.txt"), "from-src")
	writeTextFile(t, filepath.Join(dst, "x.txt"), "from-dst")

	err := moveDirContents(src, dst)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !strings.Contains(err.Error(), "目标路径已存在") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceInFilesMatcherAndSkipGit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTextFile(t, filepath.Join(root, "main.go"), "package main\n// app_layout\n")
	writeTextFile(t, filepath.Join(root, "conf.proto"), "package app.app_layout;\n")
	writeTextFile(t, filepath.Join(root, ".git", "tracked.go"), "app_layout\n")
	if err := os.Symlink("main.go", filepath.Join(root, "main_link.go")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	if err := replaceInFiles(root, "app_layout", "order_api", onlyProtoFiles); err != nil {
		t.Fatalf("replaceInFiles(proto) returned error: %v", err)
	}
	if got := mustReadFile(t, filepath.Join(root, "conf.proto")); !strings.Contains(got, "order_api") {
		t.Fatalf("proto file should be replaced: %s", got)
	}
	if got := mustReadFile(t, filepath.Join(root, "main.go")); !strings.Contains(got, "app_layout") {
		t.Fatalf("go file should not change in proto-only replace: %s", got)
	}

	if err := replaceInFiles(root, "app_layout", "order-api", nil); err != nil {
		t.Fatalf("replaceInFiles(all) returned error: %v", err)
	}
	if got := mustReadFile(t, filepath.Join(root, "main.go")); !strings.Contains(got, "order-api") {
		t.Fatalf("go file should be replaced in all-files mode: %s", got)
	}
	if got := mustReadFile(t, filepath.Join(root, ".git", "tracked.go")); strings.Contains(got, "order-api") {
		t.Fatalf(".git files should be skipped: %s", got)
	}
}

func TestFetchLayoutByFileURLAndFailure(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 不可用，跳过 fetchLayout file:// 分支测试")
	}

	srcRepo := t.TempDir()
	createLayoutFixture(t, srcRepo)
	gitInitAndCommit(t, srcRepo)

	layoutRoot, cleanup, err := fetchLayout("file://" + srcRepo)
	if err != nil {
		t.Fatalf("fetchLayout(file://) returned error: %v", err)
	}
	if !pathExists(filepath.Join(layoutRoot, "app", "app_layout", "cmd", "main.go")) {
		t.Fatalf("cloned layout does not contain expected template files")
	}
	cleanup()

	missingRepo := "file://" + filepath.Join(t.TempDir(), "not-exists-repo")
	_, _, err = fetchLayout(missingRepo)
	if err == nil {
		t.Fatalf("expected fetchLayout() failure for missing repo")
	}
	if !strings.Contains(err.Error(), "拉取 layout 失败") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureEmptyDirWithFilePath(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "target")
	writeTextFile(t, filePath, "not-a-dir")

	err := ensureEmptyDir(filePath)
	if err == nil {
		t.Fatalf("expected ensureEmptyDir() to fail on file path")
	}
	if !strings.Contains(err.Error(), "输出路径不是目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateInMonoRepoTargetExists(t *testing.T) {
	t.Parallel()

	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	outputDir := t.TempDir()
	appRoot := filepath.Join(outputDir, "app")
	mustMkdirAll(t, filepath.Join(appRoot, "order-api"))

	err := createInMonoRepo(outputDir, "github.com/acme/mono", "order-api", filepath.Join(layoutDir, templateSubDir))
	if err == nil {
		t.Fatalf("expected createInMonoRepo() to fail when target exists")
	}
	if !strings.Contains(err.Error(), "目录已存在") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateInSingleRepoMissingTemplatePath(t *testing.T) {
	t.Parallel()

	layoutRoot := t.TempDir()
	writeTextFile(t, filepath.Join(layoutRoot, "go.mod"), "module "+templateModule+"\n")
	outputDir := filepath.Join(t.TempDir(), "out")

	err := createInSingleRepo(outputDir, "github.com/acme/order", "order-api", layoutRoot, false)
	if err == nil {
		t.Fatalf("expected createInSingleRepo() to fail when app/app_layout missing")
	}
	if !strings.Contains(err.Error(), "单仓模式需要模板中存在目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyDirAndReplaceFunctionsErrorPaths(t *testing.T) {
	t.Parallel()

	if err := copyDir(filepath.Join(t.TempDir(), "missing-src"), filepath.Join(t.TempDir(), "dst")); err == nil {
		t.Fatalf("expected copyDir() to fail on missing source")
	}

	if err := moveDirContents(filepath.Join(t.TempDir(), "missing-src"), t.TempDir()); err == nil {
		t.Fatalf("expected moveDirContents() to fail on missing source")
	}

	if err := replaceInFiles(filepath.Join(t.TempDir(), "missing-root"), "x", "y", nil); err == nil {
		t.Fatalf("expected replaceInFiles() to fail on missing root")
	}

	if err := copyFile(filepath.Join(t.TempDir(), "missing.txt"), filepath.Join(t.TempDir(), "out", "a.txt")); err == nil {
		t.Fatalf("expected copyFile() to fail on missing source")
	}

	if err := replaceInFile(filepath.Join(t.TempDir(), "missing.txt"), "x", "y"); err == nil {
		t.Fatalf("expected replaceInFile() to fail on missing file")
	}
}

func TestReplaceInFileNoMatch(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "a.txt")
	writeTextFile(t, file, "hello world")
	if err := replaceInFile(file, "app_layout", "order-api"); err != nil {
		t.Fatalf("replaceInFile() returned error: %v", err)
	}
	if got := mustReadFile(t, file); got != "hello world" {
		t.Fatalf("file should stay unchanged when no match, got: %s", got)
	}
}

func TestFetchLayoutWithoutGitInPath(t *testing.T) {
	t.Setenv("PATH", "")
	_, cleanup, err := fetchLayout("https://example.com/fake/layout.git")
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatalf("expected fetchLayout() to fail when git is unavailable")
	}
	if !strings.Contains(err.Error(), "未找到 git") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSingleRepoExecutesPostTasks(t *testing.T) {
	layoutDir := t.TempDir()
	createLayoutFixture(t, layoutDir)

	binDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "task.log")
	taskScript := strings.TrimSpace(`
#!/usr/bin/env bash
set -e
echo "$*" >> "$TASK_LOG"
`) + "\n"
	writeTextFile(t, filepath.Join(binDir, "task"), taskScript)
	if err := os.Chmod(filepath.Join(binDir, "task"), 0o755); err != nil {
		t.Fatalf("chmod fake task failed: %v", err)
	}

	t.Setenv("TASK_LOG", logFile)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	outputBaseDir := t.TempDir()
	err := Run("github.com/acme/order", "order-api", Options{
		LayoutSource: layoutDir,
		OutputDir:    outputBaseDir,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	logContent := mustReadFile(t, logFile)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 task calls, got %d: %q", len(lines), logContent)
	}
	if lines[0] != "conf" {
		t.Fatalf("expected first task call to be 'conf', got: %q", lines[0])
	}
	if lines[1] != "wire -- ." {
		t.Fatalf("expected second task call to be 'wire -- .', got: %q", lines[1])
	}
}

func createLayoutFixture(t *testing.T, layoutDir string) {
	t.Helper()

	writeTextFile(t, filepath.Join(layoutDir, "go.mod"), "module "+templateModule+"\n")
	writeTextFile(t, filepath.Join(layoutDir, ".golangci.yml"), "gci:\n  sections:\n    - prefix("+templateModule+")\n")
	writeTextFile(t, filepath.Join(layoutDir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeTextFile(t, filepath.Join(layoutDir, "app", "app_layout", "cmd", "main.go"), strings.TrimSpace(`
package main

import _ "`+templateModule+`/app/app_layout/internal/conf"

func main() {}
`)+"\n")
	writeTextFile(t, filepath.Join(layoutDir, "app", "app_layout", "internal", "conf", "conf.proto"), strings.TrimSpace(`
syntax = "proto3";
package app.app_layout;
message Bootstrap {}
`)+"\n")
	writeTextFile(t, filepath.Join(layoutDir, "app", "app_layout", "configs", "config.yaml"), "name: app_layout\n")
}

func writeTextFile(t *testing.T, path, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s failed: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s failed: %v", path, err)
	}
	return string(data)
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s failed: %v", path, err)
	}
}

func gitInitAndCommit(t *testing.T, repoDir string) {
	t.Helper()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "commit.gpgsign", "false")
	runGit(t, repoDir, "config", "user.email", "tester@example.com")
	runGit(t, repoDir, "config", "user.name", "tester")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, strings.TrimSpace(string(out)))
	}
}
