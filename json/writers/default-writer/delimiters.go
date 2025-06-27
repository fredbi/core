package writer

// Comma writes a comma separator, ','.
func (w *W) Comma() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte(',')
	w.err = w.buffer.Err()
}

// Colon writes a colon separator, ':'.
func (w *W) Colon() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte(':')
	w.err = w.buffer.Err()
}

// EndArray writes an end-of-array separator, i.e. ']'.
func (w *W) EndArray() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte(']')
	w.err = w.buffer.Err()
}

// EndObject writes an end-of-object separator, i.e. '}'.
func (w *W) EndObject() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte('}')
	w.err = w.buffer.Err()
}

// StartArray writes a start-of-array separator, i.e. '['.
func (w *W) StartArray() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte('[')
	w.err = w.buffer.Err()
}

// StartObject writes a start-of-object separator, i.e. '{'.
func (w *W) StartObject() {
	if w.err != nil {
		return
	}

	w.buffer.WriteSingleByte('{')
	w.err = w.buffer.Err()
}
