package editor

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/jexme/scrawl/internal/collection"
	"github.com/jexme/scrawl/internal/i18n"
	"github.com/jexme/scrawl/internal/settings"
)

// renderCurrent renders the selection with annotations to PNG.
func (m *monitor) renderCurrent() ([]byte, *image.RGBA, error) {
	if m.editing {
		m.commitText()
	}
	img, err := render(m.frame.Img, m.sel, m.shapes, m.shaper)
	if err != nil {
		return nil, nil, fmt.Errorf(i18n.T("render: %w"), err)
	}
	data, err := encodePNG(img)
	if err != nil {
		return nil, nil, err
	}
	return data, img, nil
}

// addToCollection puts the selection into the collection and closes the overlay: the next
// launch will show the collection bar.
func (m *monitor) addToCollection() {
	if !m.hasSel {
		notify(i18n.T("Collection"), i18n.T("Select an area first"))
		return
	}
	data, _, err := m.renderCurrent()
	if err != nil {
		fail(i18n.T("collection"), err)
		return
	}
	if _, err := collection.Add(data); err != nil {
		fail(i18n.T("collection"), err)
		return
	}
	m.inform(i18n.T("Added to the collection"), fmt.Sprintf(i18n.T("Images in the collection: %d"), collection.Count()))
	m.quit()
}

// collected returns the collection images and, if there is a selection, the current
// area as the last one.
func (m *monitor) collected() (paths []string, current []byte, imgs []*image.RGBA, err error) {
	paths, err = collection.List()
	if err != nil {
		return nil, nil, nil, err
	}
	imgs, err = loadPNGs(paths)
	if err != nil {
		return nil, nil, nil, err
	}
	if m.hasSel {
		data, img, err := m.renderCurrent()
		if err != nil {
			return nil, nil, nil, err
		}
		current = data
		imgs = append(imgs, img)
	}
	return paths, current, imgs, nil
}

// glueCollection glues the collection (and the current area, if one is selected) into
// one image, saves it, copies it to the clipboard and clears the collection.
func (m *monitor) glueCollection() {
	_, _, imgs, err := m.collected()
	if err != nil {
		fail(i18n.T("glue"), err)
		return
	}
	if len(imgs) < 2 {
		notify(i18n.T("Collage"), i18n.T("At least two images are needed: add to the collection or select an area"))
		return
	}
	cfg := m.store.Get()
	out, err := compose(imgs, cfg.Glue.Labels(len(imgs)), m.shaper, cfg.Glue)
	if err != nil {
		fail(i18n.T("glue"), err)
		return
	}
	data, err := encodePNG(out)
	if err != nil {
		fail(i18n.T("glue"), err)
		return
	}
	path, err := saveFile(m.store.Get(), data)
	if err != nil {
		fail(i18n.T("glue"), err)
		return
	}
	msg := path
	if err := copyPNG(data); err != nil {
		fail(i18n.T("clipboard"), err)
	} else {
		msg += i18n.T(" (and to the clipboard)")
	}
	if err := collection.Clear(); err != nil {
		fail(i18n.T("clearing the collection"), err)
	}
	m.inform(fmt.Sprintf(i18n.T("Images glued: %d"), len(imgs)), msg)
	m.quit()
}

// sendSeparately saves the collection images (and the current area) as separate
// files and puts the file list on the clipboard: in a chat they are pasted as
// separate attachments.
func (m *monitor) sendSeparately() {
	paths, current, _, err := m.collected()
	if err != nil {
		fail(i18n.T("separate files"), err)
		return
	}
	var datas [][]byte
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			fail(i18n.T("separate files"), err)
			return
		}
		datas = append(datas, data)
	}
	if current != nil {
		datas = append(datas, current)
	}
	if len(datas) == 0 {
		notify(i18n.T("Separate files"), i18n.T("The collection is empty: select an area or add images"))
		return
	}

	// Files of one set differ by a suffix, the name is taken from the current time.
	stamp := time.Now().Format(settings.StampLayout)
	var saved []string
	for i, data := range datas {
		path, err := saveNumbered(m.store.Get(), data, stamp, i+1)
		if err != nil {
			fail(i18n.T("separate files"), err)
			return
		}
		saved = append(saved, path)
	}
	msg := fmt.Sprintf(i18n.T("%d file(s) in %s"), len(saved), filepath.Dir(saved[0]))
	if err := copyFiles(saved); err != nil {
		fail(i18n.T("clipboard"), err)
	} else {
		msg += i18n.T(", list is in the clipboard")
	}
	if err := collection.Clear(); err != nil {
		fail(i18n.T("clearing the collection"), err)
	}
	m.inform(i18n.T("Screenshots are ready"), msg)
	m.quit()
}

// clearCollection clears the collection and updates the bar.
func (m *monitor) clearCollection() {
	if err := collection.Clear(); err != nil {
		fail(i18n.T("clearing the collection"), err)
		return
	}
	m.queued = 0
	m.inform(i18n.T("Collection cleared"), "")
}
