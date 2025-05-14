package store

import (
	"io"
)

func (s *Store) compressRatioHeuristic(size int) int {
	const minCompressedSize = 10 // flate never produces compressed strings smaller then 9 bytes
	return max(size/s.compressionLevel, minCompressedSize)
}

func (s *Store) uncompressRatioHeuristic(size int) int {
	return size * s.compressionLevel
}

func (s *Store) compressString(value []byte, buffer ...[]byte) []byte {
	wrt, redeemWriter := borrowBufferWithRedeem(s.compressRatioHeuristic(len(value)))
	defer redeemWriter()

	deflater := s.cw
	s.cw.Reset(wrt)

	_, err := deflater.Write(value)
	assertCompressDeflateError(err)
	_ = deflater.Close()

	out := ensureEmptyBuffer(wrt.Len(), buffer...)
	out = append(out, wrt.Bytes()...) // cannot return the slice from wrt, which becomes unusable after the parent writer is redeemed

	return out
}

func (s *Store) uncompressString(value []byte, buffer ...[]byte) []byte {
	rdr, redeemReader := borrowBufferWithRedeem(len(value)) // use [bytes.Buffer] rather than [bytes.Reader] because it may be recycled
	_, _ = rdr.Write(value)                                 // since we don't use [bytes.Reader] we indulge into an extra copy: TODO modified version of bytes.Buffer
	wrt, redeemWriter := borrowBufferWithRedeem(s.uncompressRatioHeuristic(len(value)))
	defer redeemWriter()

	inflater, redeemInflater := borrowFlateReaderWithRedeem(rdr, s.dict)

	_, err := io.Copy(wrt, inflater)
	assertCompressInflateError(err)
	_ = inflater.Close()

	redeemInflater()
	redeemReader()

	out := ensureEmptyBuffer(wrt.Len(), buffer...)
	out = append(out, wrt.Bytes()...) // cannot return the slice from wrt, which becomes unusable after the parent writer is redeemed

	return out
}

func (s *Store) uncompressStringReader(value []byte) (io.Reader, func()) {
	rdr, redeemReader := borrowBufferWithRedeem(len(value)) // use [bytes.Buffer] rather than [bytes.Reader] because it may be recycled
	_, _ = rdr.Write(value)                                 // since we don't use [bytes.Reader] we indulge into an extra copy
	inflater, redeemInflater := borrowFlateReaderWithRedeem(rdr, s.dict)

	return rdr, func() {
		_ = inflater.Close()
		redeemInflater()
		redeemReader()
	}
}

// TODO: simplistic version of a bytes.Buffer that knows how to Read and ReadByte from a byte slice
// perhaps in the pools package
