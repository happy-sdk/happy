// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/happy-sdk/happy"
)

func (r *releaser) Initialize(sess *happy.Session, path string) error {
	config, err := newConfiguration(sess, path)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.config = *config
	r.sess = sess
	r.mu.Unlock()
	sess.Log().Ok("releaser initialized", slog.String("wd", config.WD))
	return nil
}

func (r *releaser) Run(next string) error {
	if err := r.confirmConfig(next); err != nil {
		return err
	}
	return nil
}

func (r *releaser) session() (*happy.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.sess == nil {
		return nil, fmt.Errorf("releaser not initialized with session")
	}
	return r.sess, nil
}

func (r *releaser) confirmConfig(next string) error {
	sess, err := r.session()
	if err != nil {
		return err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.sess.Set("releaser.next", next); err != nil {
		return err
	}

	m, err := r.config.getConfirmConfigModel(sess)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	var userSelectedYes bool
	if model, err := p.Run(); err != nil {
		return fmt.Errorf("Error running program: %w", err)
	} else {
		m, ok := model.(configTable)
		if !ok {
			return fmt.Errorf("Could not assert model type.")
		}
		userSelectedYes = m.yes
	}
	if !userSelectedYes {
		return fmt.Errorf("release canceled by user.")
	}
	return nil
}
