package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"

	"gioui.org/app"
)

func getCacheDir() (string, error) {
	if runtime.GOOS == "android" && app.ID != "" {
		return path.Join("/data/data", app.ID, "cache"), nil
	}

	if cache, err := os.UserCacheDir(); err == nil {
		return path.Join(cache, "dc-menu"), nil
	} else {
		return "", err
	}
}

// get the path to put the dining common's cache file
func dcCacheLocation(dc string) (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(cacheDir, dc), nil
}

func loadFromCache(dc string) (Menu, error) {
	if dc == "" {
		return Menu{}, fmt.Errorf("refusing to load cache with empty dc string")
	}

	cacheFile, err := dcCacheLocation(dc)
	if err != nil {
		return Menu{}, err
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return Menu{}, err
	}

	var menu Menu
	err = json.Unmarshal(data, &menu)
	return menu, err
}

func saveToCache(dc string, menu Menu) error {
	if dc == "" {
		return fmt.Errorf("refusing to cache with empty dc string")
	}

	cacheFile, err := dcCacheLocation(dc)
	if err != nil {
		return err
	}

	data, err := json.Marshal(menu)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(cacheFile), 0750)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0640)
}
