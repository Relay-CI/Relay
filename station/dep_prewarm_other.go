//go:build !linux

package main

// depCacheKey, depCacheRestore, and depCacheSave are no-ops on non-Linux
// platforms.  The dep prewarm cache requires Linux filesystem primitives
// (hard links across ext4) and namespace isolation for installs.

func depCacheKey(_, _, _ string) (key, artifactDir string, ok bool) { return "", "", false }
func depCacheRestore(_, _, _, _ string) bool                         { return false }
func depCacheSave(_, _, _, _ string)                                 { /* no-op */ }
