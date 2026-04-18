package rns

func makeHashKey(hash []byte) (hashKey, bool) {
	if len(hash) < truncatedHashBytes {
		return hashKey{}, false
	}
	var key hashKey
	copy(key[:], hash[:truncatedHashBytes])
	return key, true
}
