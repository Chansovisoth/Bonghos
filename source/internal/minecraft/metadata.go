package minecraft

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ServerMetadata describes the game and loader versions installed in a server
// pack. Values are taken from pack-owned metadata first, then from the
// loader's installed artifacts.
type ServerMetadata struct {
	MinecraftVersion string `json:"minecraft_version"`
	Modloader        string `json:"modloader"`
	ModloaderVersion string `json:"modloader_version"`
}

var (
	metadataAssignmentRE = regexp.MustCompile(`(?mi)^\s*(?:(?:export|set)\s+)?["']?([A-Z][A-Z0-9_]*)\s*=\s*["']?([^"'\r\n#\s]+)`)
	fabricServerJarRE    = regexp.MustCompile(`(?i)^fabric-server-mc\.([^-]+)-loader\.([^-]+)-launcher\..+\.jar$`)
	forgeArtifactRE      = regexp.MustCompile(`^(.+)-([0-9]+(?:\.[0-9]+)+(?:[-+._][A-Za-z0-9.-]+)?)$`)
)

type curseForgeManifest struct {
	Minecraft struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID      string `json:"id"`
			Primary bool   `json:"primary"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
}

type modrinthIndex struct {
	Dependencies map[string]string `json:"dependencies"`
}

// DetectServerMetadata detects CurseForge, Forge, NeoForge, Fabric, Quilt and
// vanilla server versions without modifying the server pack.
func DetectServerMetadata(root string) (ServerMetadata, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ServerMetadata{}, err
	}
	meta := detectCurseForgeManifest(filepath.Join(root, "manifest.json"))
	mergeMissingMetadata(&meta, detectModrinthIndex(filepath.Join(root, "modrinth.index.json")))

	for _, name := range metadataTextFiles(root) {
		data, readErr := os.ReadFile(name)
		if readErr != nil || len(data) > 1<<20 {
			continue
		}
		applyMetadataAssignments(&meta, string(data))
	}

	detectFabricLauncherName(root, &meta)
	detectInstalledLoaderArtifacts(root, &meta)
	if meta.MinecraftVersion == "" {
		meta.MinecraftVersion = detectMinecraftJarVersion(root)
	}
	if meta.MinecraftVersion == "" && meta.Modloader == "neoforge" {
		meta.MinecraftVersion = minecraftVersionFromNeoForge(meta.ModloaderVersion)
	}
	return meta, nil
}

func detectModrinthIndex(name string) ServerMetadata {
	data, err := os.ReadFile(name)
	if err != nil || len(data) > 2<<20 {
		return ServerMetadata{}
	}
	var index modrinthIndex
	if json.Unmarshal(data, &index) != nil {
		return ServerMetadata{}
	}
	meta := ServerMetadata{MinecraftVersion: strings.TrimSpace(index.Dependencies["minecraft"])}
	for _, loader := range []struct{ dependency, name string }{
		{"neoforge", "neoforge"},
		{"forge", "forge"},
		{"fabric-loader", "fabric"},
		{"quilt-loader", "quilt"},
	} {
		if version := strings.TrimSpace(index.Dependencies[loader.dependency]); version != "" {
			meta.Modloader = loader.name
			meta.ModloaderVersion = version
			break
		}
	}
	return meta
}

func mergeMissingMetadata(dst *ServerMetadata, src ServerMetadata) {
	if dst.MinecraftVersion == "" {
		dst.MinecraftVersion = src.MinecraftVersion
	}
	if dst.Modloader == "" {
		dst.Modloader = src.Modloader
	}
	if dst.ModloaderVersion == "" {
		dst.ModloaderVersion = src.ModloaderVersion
	}
}

func detectCurseForgeManifest(name string) ServerMetadata {
	data, err := os.ReadFile(name)
	if err != nil || len(data) > 2<<20 {
		return ServerMetadata{}
	}
	var manifest curseForgeManifest
	if json.Unmarshal(data, &manifest) != nil {
		return ServerMetadata{}
	}
	meta := ServerMetadata{MinecraftVersion: strings.TrimSpace(manifest.Minecraft.Version)}
	loaders := manifest.Minecraft.ModLoaders
	for _, loader := range loaders {
		if loader.Primary {
			applyCurseForgeLoaderID(&meta, loader.ID)
			return meta
		}
	}
	if len(loaders) > 0 {
		applyCurseForgeLoaderID(&meta, loaders[0].ID)
	}
	return meta
}

func applyCurseForgeLoaderID(meta *ServerMetadata, id string) {
	id = strings.TrimSpace(id)
	lower := strings.ToLower(id)
	for _, loader := range []string{"neoforge", "forge", "fabric", "quilt"} {
		prefix := loader + "-"
		if strings.HasPrefix(lower, prefix) {
			meta.Modloader = loader
			meta.ModloaderVersion = strings.TrimSpace(id[len(prefix):])
			return
		}
	}
}

func metadataTextFiles(root string) []string {
	names := []string{"variables.txt", ".env"}
	for name := range knownStartupNames {
		names = append(names, name)
	}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(root, name))
	}
	return paths
}

func applyMetadataAssignments(meta *ServerMetadata, body string) {
	values := make(map[string]string)
	for _, match := range metadataAssignmentRE.FindAllStringSubmatch(body, -1) {
		values[strings.ToUpper(match[1])] = strings.TrimSpace(match[2])
	}
	first := func(keys ...string) string {
		for _, key := range keys {
			if value := values[key]; value != "" {
				return value
			}
		}
		return ""
	}
	if meta.MinecraftVersion == "" {
		meta.MinecraftVersion = first("MINECRAFT_VERSION", "MC_VERSION")
	}
	if meta.Modloader == "" {
		meta.Modloader = normalizeModloader(first("MODLOADER", "MOD_LOADER"))
	}
	if meta.ModloaderVersion == "" {
		meta.ModloaderVersion = first("MODLOADER_VERSION", "MOD_LOADER_VERSION")
	}
	if meta.Modloader == "" {
		for _, candidate := range []struct{ loader, key string }{
			{"neoforge", "NEOFORGE_VERSION"},
			{"forge", "FORGE_VERSION"},
			{"fabric", "FABRIC_LOADER_VERSION"},
			{"quilt", "QUILT_LOADER_VERSION"},
		} {
			if version := values[candidate.key]; version != "" {
				meta.Modloader = candidate.loader
				if meta.ModloaderVersion == "" {
					meta.ModloaderVersion = version
				}
				break
			}
		}
	}
}

func detectFabricLauncherName(root string, meta *ServerMetadata) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := fabricServerJarRE.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		if meta.MinecraftVersion == "" {
			meta.MinecraftVersion = match[1]
		}
		if meta.Modloader == "" {
			meta.Modloader = "fabric"
		}
		if meta.ModloaderVersion == "" {
			meta.ModloaderVersion = match[2]
		}
		return
	}
}

func detectInstalledLoaderArtifacts(root string, meta *ServerMetadata) {
	type artifact struct {
		loader string
		glob   string
	}
	artifacts := []artifact{
		{"neoforge", filepath.Join(root, "libraries", "net", "neoforged", "neoforge", "*")},
		{"forge", filepath.Join(root, "libraries", "net", "minecraftforge", "forge", "*")},
		{"fabric", filepath.Join(root, "libraries", "net", "fabricmc", "fabric-loader", "*")},
		{"quilt", filepath.Join(root, "libraries", "org", "quiltmc", "quilt-loader", "*")},
	}
	for _, artifact := range artifacts {
		matches, _ := filepath.Glob(artifact.glob)
		if len(matches) == 0 {
			continue
		}
		version := filepath.Base(matches[len(matches)-1])
		minecraftVersion := ""
		if artifact.loader == "forge" {
			if match := forgeArtifactRE.FindStringSubmatch(version); match != nil {
				minecraftVersion, version = match[1], match[2]
			}
		}
		if meta.Modloader == "" {
			meta.Modloader = artifact.loader
		}
		if meta.Modloader == artifact.loader && meta.ModloaderVersion == "" {
			meta.ModloaderVersion = version
		}
		if meta.MinecraftVersion == "" && minecraftVersion != "" {
			meta.MinecraftVersion = minecraftVersion
		}
		return
	}
}

func detectMinecraftJarVersion(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if entry.IsDir() || !strings.HasSuffix(name, ".jar") ||
			!(name == "server.jar" || strings.HasPrefix(name, "minecraft_server")) {
			continue
		}
		reader, openErr := zip.OpenReader(filepath.Join(root, entry.Name()))
		if openErr != nil {
			continue
		}
		version := versionFromJar(reader)
		reader.Close()
		if version != "" {
			return version
		}
	}
	return ""
}

func versionFromJar(reader *zip.ReadCloser) string {
	for _, file := range reader.File {
		if file.Name != "version.json" {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return ""
		}
		var version struct {
			ID string `json:"id"`
		}
		err = json.NewDecoder(stream).Decode(&version)
		stream.Close()
		if err == nil {
			return strings.TrimSpace(version.ID)
		}
	}
	return ""
}

func minecraftVersionFromNeoForge(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	if major >= 26 {
		return parts[0] + "." + parts[1]
	}
	return "1." + parts[0] + "." + parts[1]
}
