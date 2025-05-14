package writer

func (w *W) Comma() {
	w.buffer.AppendByte(',')
}

func (w *W) EndArray() {
	w.buffer.AppendByte(']')
}

func (w *W) EndObject() {
	w.buffer.AppendByte('}')
}

func (w *W) StartObject() {
	w.buffer.AppendByte('{')
}

func (w *W) StartArray() {
	w.buffer.AppendByte('[')
}
