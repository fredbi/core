package formats

// DeepCopyInto copies the receiver and writes its value into out.
func (d Date) DeepCopyInto(out *Date) {
	*out = d
}

// DeepCopy copies the receiver into a new [Date].
func (d Date) DeepCopy() *Date {
	out := NewDate()
	d.DeepCopyInto(out)

	return out
}

// DeepCopyInto copies the receiver and writes its value into out.
func (d *Duration) DeepCopyInto(out *Duration) {
	*out = *d
}

// DeepCopy copies the receiver into a new Duration.
func (d *Duration) DeepCopy() *Duration {
	if d == nil {
		return nil
	}
	out := new(Duration)
	d.DeepCopyInto(out)

	return out
}

// DeepCopyInto copies the receiver and writes its value into out.
func (t *DateTime) DeepCopyInto(out *DateTime) {
	*out = *t
}

// DeepCopy copies the receiver into a new DateTime.
func (t *DateTime) DeepCopy() *DateTime {
	if t == nil {
		return nil
	}
	out := new(DateTime)
	t.DeepCopyInto(out)

	return out
}
