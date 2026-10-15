package editor

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/jexme/screen-editor/internal/collection"
)

// renderCurrent рендерит выделение с аннотациями в PNG.
func (m *monitor) renderCurrent() ([]byte, *image.RGBA, error) {
	if m.editing {
		m.commitText()
	}
	img, err := render(m.frame.Img, m.sel, m.shapes, m.shaper)
	if err != nil {
		return nil, nil, fmt.Errorf("рендер: %w", err)
	}
	data, err := encodePNG(img)
	if err != nil {
		return nil, nil, err
	}
	return data, img, nil
}

// addToCollection кладёт выделение в подборку и закрывает оверлей: следующий
// запуск покажет плашку подборки.
func (m *monitor) addToCollection() {
	if !m.hasSel {
		notify("Подборка", "Сначала выделите область")
		return
	}
	data, _, err := m.renderCurrent()
	if err != nil {
		fail("подборка", err)
		return
	}
	if _, err := collection.Add(data); err != nil {
		fail("подборка", err)
		return
	}
	notify("Добавлено в подборку", fmt.Sprintf("Картинок в подборке: %d", collection.Count()))
	m.quit()
}

// collected возвращает картинки подборки и, если есть выделение, текущую
// область последней.
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

// glueCollection склеивает подборку (и текущую область, если она выделена) в
// одну картинку, сохраняет её, копирует в буфер обмена и очищает подборку.
func (m *monitor) glueCollection() {
	_, _, imgs, err := m.collected()
	if err != nil {
		fail("склейка", err)
		return
	}
	if len(imgs) < 2 {
		notify("Склейка", "Нужно минимум две картинки: добавьте в подборку или выделите область")
		return
	}
	out, err := compose(imgs, composeLabels(len(imgs)), m.shaper)
	if err != nil {
		fail("склейка", err)
		return
	}
	data, err := encodePNG(out)
	if err != nil {
		fail("склейка", err)
		return
	}
	path, err := saveFile(m.store.Get().SaveDir, data)
	if err != nil {
		fail("склейка", err)
		return
	}
	msg := path
	if err := copyPNG(data); err != nil {
		fail("буфер обмена", err)
	} else {
		msg += " (и в буфер обмена)"
	}
	if err := collection.Clear(); err != nil {
		fail("очистка подборки", err)
	}
	notify(fmt.Sprintf("Склеено картинок: %d", len(imgs)), msg)
	m.quit()
}

// sendSeparately сохраняет картинки подборки (и текущую область) отдельными
// файлами и кладёт в буфер обмена список файлов: в чат они вставятся как
// отдельные вложения.
func (m *monitor) sendSeparately() {
	paths, current, _, err := m.collected()
	if err != nil {
		fail("отдельные файлы", err)
		return
	}
	var datas [][]byte
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			fail("отдельные файлы", err)
			return
		}
		datas = append(datas, data)
	}
	if current != nil {
		datas = append(datas, current)
	}
	if len(datas) == 0 {
		notify("Отдельные файлы", "Подборка пуста: выделите область или добавьте картинки")
		return
	}

	// Файлы одного набора различаются суффиксом, имя берём по текущему времени.
	stamp := time.Now().Format("20060102-150405")
	var saved []string
	for i, data := range datas {
		path, err := saveNumbered(m.store.Get().SaveDir, data, stamp, i+1)
		if err != nil {
			fail("отдельные файлы", err)
			return
		}
		saved = append(saved, path)
	}
	msg := fmt.Sprintf("%d файла(ов) в %s", len(saved), filepath.Dir(saved[0]))
	if err := copyFiles(saved); err != nil {
		fail("буфер обмена", err)
	} else {
		msg += ", список в буфере обмена"
	}
	if err := collection.Clear(); err != nil {
		fail("очистка подборки", err)
	}
	notify("Скриншоты готовы", msg)
	m.quit()
}

// clearCollection очищает подборку и обновляет плашку.
func (m *monitor) clearCollection() {
	if err := collection.Clear(); err != nil {
		fail("очистка подборки", err)
		return
	}
	m.queued = 0
	notify("Подборка очищена", "")
}
