package main

import (
	"projetop2p/internal/community"
	"projetop2p/internal/desktop"
	"projetop2p/internal/profile"
)

func (a *App) Profiles(ids []string) ([]profile.Metadata, error) {
	if err := a.initialized(); err != nil {
		return nil, err
	}
	return a.core.Profiles(ids)
}

func (a *App) ProfileAsset(hash string) (*desktop.ImageAsset, error) {
	if err := a.initialized(); err != nil {
		return nil, err
	}
	return a.core.ProfileAsset(hash)
}

func (a *App) SaveProfile(change desktop.ProfileChange) (profile.Metadata, error) {
	if err := a.initialized(); err != nil {
		return profile.Metadata{}, err
	}
	return a.core.SaveProfile(change)
}

func (a *App) SaveCommunityArtwork(id, channel string, image *desktop.ImageAsset) (community.Artwork, error) {
	if err := a.initialized(); err != nil {
		return community.Artwork{}, err
	}
	return a.core.SaveCommunityArtwork(id, channel, image)
}
