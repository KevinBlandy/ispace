package rege

import "regexp"

// Sha256Hex sha256 哈希，不区分大消息
var Sha256Hex = regexp.MustCompile("^[a-fA-F0-9]{64}$")
