package main

/* copied from hst and must never be changed */

const (
	userOffset = 100000
	rangeSize  = userOffset / 10

	identityStart = 0
	identityEnd   = appEnd - appStart

	appStart = rangeSize * 1
	appEnd   = appStart + rangeSize - 1
)

func toUser(userid, appid uint32) uint32 { return userid*userOffset + appStart + appid }
