package _plugin

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	_function "github.com/BANKA2017/tbsign_go/functions"
	"github.com/BANKA2017/tbsign_go/model"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func init() {
	PluginList.Register(WeltolkAutoreplyPlugin)
}

// ============================================================
// Plugin struct
// ============================================================

type WeltolkAutoreplyPluginType struct {
	PluginInfo
}

var WeltolkAutoreplyPlugin = _function.VPtr(WeltolkAutoreplyPluginType{
	PluginInfo{
		Name:              "weltolk_autoreply",
		PluginNameCN:      "自动回帖",
		PluginNameCNShort: "自动回帖",
		PluginNameFE:      "weltolk_autoreply",
		Version:           "1.0",
		Options: map[string]string{
			"weltolk_autoreply_limit":        "5",
			"weltolk_autoreply_id":           "0",
			"weltolk_autoreply_action_limit": "50",
		},
		SettingOptions: map[string]PluginSettingOption{
			"weltolk_autoreply_limit": {
				OptionName:   "weltolk_autoreply_limit",
				OptionNameCN: "可添加自动回帖任务上限",
				Validate: &_function.OptionRule{
					Min: _function.VPtr(int64(0)),
				},
			},
			"weltolk_autoreply_action_limit": {
				OptionName:   "weltolk_autoreply_action_limit",
				OptionNameCN: "每分钟最大执行数",
				Validate: &_function.OptionRule{
					Min: _function.VPtr(int64(0)),
				},
			},
		},
		Endpoints: []PluginEndpointStruct{
			{Method: http.MethodGet, Path: "switch", Function: PluginWeltolkAutoreplyGetSwitch},
			{Method: http.MethodPost, Path: "switch", Function: PluginWeltolkAutoreplySwitch},
			{Method: http.MethodGet, Path: "list", Function: PluginWeltolkAutoreplyGetList},
			{Method: http.MethodPatch, Path: "list", Function: PluginWeltolkAutoreplyAddTask},
			{Method: http.MethodPut, Path: "list/:id", Function: PluginWeltolkAutoreplyEditTask},
			{Method: http.MethodDelete, Path: "list/:id", Function: PluginWeltolkAutoreplyDelTask},
			{Method: http.MethodPost, Path: "list/empty", Function: PluginWeltolkAutoreplyDelAllTasks},
			{Method: http.MethodPost, Path: "test", Function: PluginWeltolkAutoreplyTest},
			{Method: http.MethodGet, Path: "settings", Function: PluginWeltolkAutoreplyGetSettings},
			{Method: http.MethodPut, Path: "settings", Function: PluginWeltolkAutoreplySetSettings},
		},
	},
})

// ============================================================
// Protobuf encoding helpers
// ============================================================

func pbEncodeVarint(value uint64) []byte {
	var buf []byte
	for value >= 0x80 {
		buf = append(buf, byte((value&0x7F)|0x80))
		value >>= 7
	}
	buf = append(buf, byte(value&0x7F))
	return buf
}

func pbEncodeTag(fieldNumber int, wireType byte) []byte {
	return pbEncodeVarint(uint64(fieldNumber<<3) | uint64(wireType))
}

func pbEncodeString(fieldNumber int, value string) []byte {
	tag := pbEncodeTag(fieldNumber, 2)
	v := []byte(value)
	lenBytes := pbEncodeVarint(uint64(len(v)))
	return append(append(tag, lenBytes...), v...)
}

func pbEncodeInt32(fieldNumber int, value int) []byte {
	tag := pbEncodeTag(fieldNumber, 0)
	return append(tag, pbEncodeVarint(uint64(value))...)
}

func pbEncodeInt64(fieldNumber int, value int64) []byte {
	tag := pbEncodeTag(fieldNumber, 0)
	return append(tag, pbEncodeVarint(uint64(value))...)
}

func pbEncodeDouble(fieldNumber int, value float64) []byte {
	tag := pbEncodeTag(fieldNumber, 1)
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(value))
	return append(tag, buf...)
}

func pbEncodeMessage(fieldNumber int, data []byte) []byte {
	tag := pbEncodeTag(fieldNumber, 2)
	lenBytes := pbEncodeVarint(uint64(len(data)))
	return append(append(tag, lenBytes...), data...)
}

// ============================================================
// Device ID generation
// ============================================================

type autoreplyDeviceIDs struct {
	CUID         string
	CUIDGalaxy2  string
	C3Aid        string
	AndroidID    string
	SampleID     string
	ZID          string
}

func autoreplyGenerateDeviceIDs() autoreplyDeviceIDs {
	// android_id: 16 hex chars
	androidIDBytes := make([]byte, 8)
	_, _ = rand.Read(androidIDBytes)
	androidID := hex.EncodeToString(androidIDBytes)

	// UUID v4
	uuidBytes := make([]byte, 16)
	_, _ = rand.Read(uuidBytes)
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40 // version 4
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80 // variant 10
	uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(uuidBytes[0:4]),
		binary.BigEndian.Uint16(uuidBytes[4:6]),
		binary.BigEndian.Uint16(uuidBytes[6:8]),
		binary.BigEndian.Uint16(uuidBytes[8:10]),
		uuidBytes[10:16],
	)

	cuid := "baidutiebaapp" + uuid

	// cuid_galaxy2: 32 uppercase hex + "|" + 9 uppercase alphanumeric
	galaxyHex := make([]byte, 16)
	_, _ = rand.Read(galaxyHex)
	galaxyB64 := make([]byte, 8)
	_, _ = rand.Read(galaxyB64)
	galaxyB64Str := strings.ToUpper(base64URLStrip(string(galaxyB64)))
	if len(galaxyB64Str) > 9 {
		galaxyB64Str = galaxyB64Str[:9]
	}
	cuidGalaxy2 := strings.ToUpper(hex.EncodeToString(galaxyHex)) + "|" + galaxyB64Str

	// c3_aid: "A00-" + 32 uppercase hex + "-" + 8 uppercase alphanumeric
	aidHex := make([]byte, 16)
	_, _ = rand.Read(aidHex)
	aidB64 := make([]byte, 8)
	_, _ = rand.Read(aidB64)
	aidB64Str := strings.ToUpper(base64URLStrip(string(aidB64)))
	if len(aidB64Str) > 8 {
		aidB64Str = aidB64Str[:8]
	}
	c3Aid := "A00-" + strings.ToUpper(hex.EncodeToString(aidHex)) + "-" + aidB64Str

	// sample_id: 16 uppercase alphanumeric
	sampleB64 := make([]byte, 12)
	_, _ = rand.Read(sampleB64)
	sampleID := strings.ToUpper(base64URLStrip(string(sampleB64)))
	if len(sampleID) > 16 {
		sampleID = sampleID[:16]
	}

	return autoreplyDeviceIDs{
		CUID:        cuid,
		CUIDGalaxy2: cuidGalaxy2,
		C3Aid:       c3Aid,
		AndroidID:   androidID,
		SampleID:    sampleID,
		ZID:         "",
	}
}

func base64URLStrip(s string) string {
	s = strings.ReplaceAll(s, "+", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "=", "")
	return s
}

// ============================================================
// Build AddPostReqIdl protobuf binary
// ============================================================

func autoreplyBuildPostProto(bduss, stoken, tbs, fname string, fid int64, tid int64, content, showName, quoteID, replyUID, floorNum, subPostID string) []byte {
	dev := autoreplyGenerateDeviceIDs()
	timestamp := time.Now().UnixMilli()
	now := time.Now()
	eventDay := fmt.Sprintf("%d%d%d", now.Year(), int(now.Month()), now.Day())
	installTime := timestamp - 86400*30*1000

	// common
	var common []byte
	common = append(common, pbEncodeInt32(1, 2)...)
	common = append(common, pbEncodeString(2, "12.35.1.0")...)
	common = append(common, pbEncodeString(3, dev.CUID)...)
	common = append(common, pbEncodeString(5, "000000000000000")...)
	common = append(common, pbEncodeString(6, "1020031h")...)
	common = append(common, pbEncodeString(7, dev.CUIDGalaxy2)...)
	common = append(common, pbEncodeInt64(8, timestamp)...)
	common = append(common, pbEncodeString(9, "SM-G988N")...)
	common = append(common, pbEncodeString(10, bduss)...)
	common = append(common, pbEncodeString(11, tbs)...)
	common = append(common, pbEncodeInt32(12, 1)...)
	common = append(common, pbEncodeString(24, "1.0.3")...)
	common = append(common, pbEncodeString(25, "9")...)
	common = append(common, pbEncodeString(26, "samsung")...)
	common = append(common, pbEncodeString(28, "3.0.0")...)
	common = append(common, pbEncodeString(29, "")...)
	common = append(common, pbEncodeString(30, stoken)...)
	common = append(common, pbEncodeString(31, dev.ZID)...)
	common = append(common, pbEncodeString(32, dev.CUIDGalaxy2)...)
	common = append(common, pbEncodeString(33, "")...)
	common = append(common, pbEncodeString(34, "")...)
	common = append(common, pbEncodeString(35, dev.C3Aid)...)
	common = append(common, pbEncodeString(36, dev.SampleID)...)
	common = append(common, pbEncodeInt32(37, 720)...)
	common = append(common, pbEncodeInt32(38, 1280)...)
	common = append(common, pbEncodeDouble(39, 1.5)...)
	common = append(common, pbEncodeInt32(40, 0)...)
	common = append(common, pbEncodeInt32(41, 0)...)
	common = append(common, pbEncodeString(42, "2.34.0")...)
	common = append(common, pbEncodeString(43, "3340042")...)
	common = append(common, pbEncodeString(44, "1038000")...)
	common = append(common, pbEncodeInt64(49, installTime)...)
	common = append(common, pbEncodeInt64(50, installTime)...)
	common = append(common, pbEncodeInt64(51, installTime)...)
	common = append(common, pbEncodeString(53, eventDay)...)
	common = append(common, pbEncodeString(54, dev.AndroidID)...)
	common = append(common, pbEncodeInt32(55, 1)...)
	common = append(common, pbEncodeString(56, "")...)
	common = append(common, pbEncodeInt32(57, 1)...)
	common = append(common, pbEncodeString(60, "0")...)
	common = append(common, pbEncodeString(61, "")...)
	common = append(common, pbEncodeString(62, "tieba/12.35.1.0")...)
	common = append(common, pbEncodeInt32(63, 1)...)
	common = append(common, pbEncodeString(70, "0.4")...)

	// data
	var data []byte
	data = append(data, pbEncodeMessage(1, common)...)
	data = append(data, pbEncodeString(6, "1")...)         // anonymous
	data = append(data, pbEncodeString(7, "0")...)         // can_no_forum
	data = append(data, pbEncodeString(8, "0")...)         // is_feedback
	data = append(data, pbEncodeString(9, "0")...)         // takephoto_num
	data = append(data, pbEncodeString(10, "0")...)        // entrance_type
	data = append(data, pbEncodeString(16, "12")...)       // vcode_tag
	data = append(data, pbEncodeString(18, "1")...)        // new_vcode
	data = append(data, pbEncodeString(19, content)...)    // content
	data = append(data, pbEncodeString(26, strconv.FormatInt(fid, 10))...) // fid
	if quoteID == "" {
		data = append(data, pbEncodeString(28, "")...) // v_fid
		data = append(data, pbEncodeString(29, "")...) // v_fname
	}
	data = append(data, pbEncodeString(30, fname)...)  // kw
	data = append(data, pbEncodeString(31, "0")...)    // is_barrage
	if quoteID == "" {
		data = append(data, pbEncodeString(32, "0")...) // barrage_time
	}
	data = append(data, pbEncodeString(45, strconv.FormatInt(tid, 10))...) // tid
	if quoteID != "" {
		data = append(data, pbEncodeString(46, quoteID)...)   // quote_id
	}
	data = append(data, pbEncodeString(47, "0")...)        // is_twzhibo_thread
	data = append(data, pbEncodeString(48, floorNum)...)   // floor_num
	if quoteID != "" {
		data = append(data, pbEncodeString(49, quoteID)...)   // repostid
	}
	if subPostID != "" {
		data = append(data, pbEncodeString(50, subPostID)...) // sub_post_id
	}
	data = append(data, pbEncodeString(51, "0")...) // is_ad
	data = append(data, pbEncodeString(52, "0")...) // is_addition
	data = append(data, pbEncodeString(53, "0")...) // is_giftpost
	// field 55 post_from
	if quoteID == "" && subPostID == "" {
		data = append(data, pbEncodeString(55, "13")...) // 主题回复
	} else if subPostID == "" {
		data = append(data, pbEncodeString(55, "0")...) // 楼层回复
	}
	// sub_post_id 非空时不编码 field 55
	data = append(data, pbEncodeString(58, showName)...) // name_show
	data = append(data, pbEncodeString(60, "0")...)      // is_pictxt
	if quoteID != "" && replyUID != "" {
		data = append(data, pbEncodeString(20, replyUID)...) // reply_uid
	}
	data = append(data, pbEncodeInt32(64, 0)...) // show_custom_figure
	data = append(data, pbEncodeInt32(67, 0)...) // is_show_bless

	// req
	req := pbEncodeMessage(1, data)
	return req
}

// ============================================================
// Protobuf response parser
// ============================================================

type autoreplyParseResult struct {
	ErrorNo    int
	ErrorMsg   string
	NeedVcode  bool
	RawDebug   string
}

func autoreplyReadVarint(data []byte, pos int) (uint64, int, bool) {
	var value uint64
	var shift uint
	for pos < len(data) {
		b := data[pos]
		pos++
		value |= uint64(b&0x7F) << shift
		if (b & 0x80) == 0 {
			return value, pos, true
		}
		shift += 7
	}
	return 0, pos, false
}

func autoreplySkipField(data []byte, pos int, wireType byte) (int, bool) {
	switch wireType {
	case 0:
		_, newPos, ok := autoreplyReadVarint(data, pos)
		return newPos, ok
	case 1:
		return pos + 8, true
	case 2:
		length, newPos, ok := autoreplyReadVarint(data, pos)
		if !ok {
			return pos, false
		}
		return newPos + int(length), true
	case 5:
		return pos + 4, true
	default:
		return pos, false
	}
}

func autoreplyParseError(data []byte) (int, string) {
	errorNo := 0
	errMsg := ""
	pos := 0
	for pos < len(data) {
		tag, newPos, ok := autoreplyReadVarint(data, pos)
		if !ok {
			break
		}
		pos = newPos
		fieldNumber := int(tag >> 3)
		wireType := byte(tag & 0x07)

		if wireType == 0 && fieldNumber == 1 {
			val, newPos2, ok := autoreplyReadVarint(data, pos)
			if !ok {
				break
			}
			errorNo = int(val)
			pos = newPos2
		} else if wireType == 2 && fieldNumber == 2 {
			length, newPos2, ok := autoreplyReadVarint(data, pos)
			if !ok {
				break
			}
			pos = newPos2
			if pos+int(length) <= len(data) {
				errMsg = string(data[pos : pos+int(length)])
			}
			pos += int(length)
		} else {
			newPos, ok := autoreplySkipField(data, pos, wireType)
			if !ok {
				break
			}
			pos = newPos
		}
	}
	return errorNo, errMsg
}

func autoreplyParseNeedVcode(data []byte) bool {
	needVcode := false
	pos := 0
	for pos < len(data) {
		tag, newPos, ok := autoreplyReadVarint(data, pos)
		if !ok {
			break
		}
		pos = newPos
		fieldNumber := int(tag >> 3)
		wireType := byte(tag & 0x07)

		if wireType == 0 {
			_, newPos, ok := autoreplyReadVarint(data, pos)
			if !ok {
				break
			}
			pos = newPos
		} else if wireType == 2 {
			length, newPos, ok := autoreplyReadVarint(data, pos)
			if !ok {
				break
			}
			pos = newPos

			if fieldNumber == 14 {
				// PostAntiInfo message
				subData := data[pos : pos+int(length)]
				sPos := 0
				for sPos < len(subData) {
					sTag, sNewPos, ok := autoreplyReadVarint(subData, sPos)
					if !ok {
						break
					}
					sPos = sNewPos
					sField := int(sTag >> 3)
					sWire := byte(sTag & 0x07)

					if sWire == 2 && sField == 3 {
						sLen, sNewPos2, ok := autoreplyReadVarint(subData, sPos)
						if ok {
							sPos = sNewPos2
							if sPos+int(sLen) <= len(subData) {
								needVcodeValue := string(subData[sPos : sPos+int(sLen)])
								needVcode = needVcodeValue != "0"
							}
						}
						break
					} else if sWire == 0 {
						_, sNewPos2, ok := autoreplyReadVarint(subData, sPos)
						if !ok {
							break
						}
						sPos = sNewPos2
					} else if sWire == 2 {
						sLen, sNewPos2, ok := autoreplyReadVarint(subData, sPos)
						if !ok {
							break
						}
						sPos = sNewPos2 + int(sLen)
					} else {
						break
					}
				}
			}

			pos += int(length)
		} else {
			break
		}
	}
	return needVcode
}

func autoreplyParseResponse(binaryData []byte) autoreplyParseResult {
	rawDebug := hex.EncodeToString(binaryData)
	if len(rawDebug) > 256 {
		rawDebug = rawDebug[:256]
	}

	errorNo := 0
	errMsg := ""
	needVcode := false

	pos := 0
	for pos < len(binaryData) {
		tag, newPos, ok := autoreplyReadVarint(binaryData, pos)
		if !ok {
			break
		}
		pos = newPos
		fieldNumber := int(tag >> 3)
		wireType := byte(tag & 0x07)

		if wireType == 0 {
			_, newPos, ok := autoreplyReadVarint(binaryData, pos)
			if !ok {
				break
			}
			pos = newPos
		} else if wireType == 1 {
			pos += 8
		} else if wireType == 2 {
			length, newPos, ok := autoreplyReadVarint(binaryData, pos)
			if !ok {
				break
			}
			pos = newPos
			if pos+int(length) <= len(binaryData) {
				subData := binaryData[pos : pos+int(length)]
				if fieldNumber == 1 {
					errorNo, errMsg = autoreplyParseError(subData)
				} else if fieldNumber == 2 {
					needVcode = autoreplyParseNeedVcode(subData)
				}
			}
			pos += int(length)
		} else if wireType == 5 {
			pos += 4
		} else {
			break
		}
	}

	return autoreplyParseResult{
		ErrorNo:   errorNo,
		ErrorMsg:  errMsg,
		NeedVcode: needVcode,
		RawDebug:  rawDebug,
	}
}

// ============================================================
// Tieba JSON API
// ============================================================

func weltolkCallTiebaJSONAPI(tid int64, bduss string, pn, rn, r string) (map[string]any, error) {
	secret := "tiebaclient!!!"
	stTime := randomInt(100, 850)
	stSize := int(float64(stTime) * (float64(randomInt(0, 2147483647))/float64(2147483647)*8 + 0.4))
	cuid := "baidutiebaapp" + strconv.Itoa(randomInt(10000000, 99999999))

	params := map[string]string{
		"_client_type":    "2",
		"_client_version": "12.41.7.1",
		"_phone_imei":     "000000000000000",
		"back":            "0",
		"cuid":            cuid,
		"floor_rn":        "3",
		"from":            "tieba",
		"kz":              strconv.FormatInt(tid, 10),
		"lz":              "0",
		"mark":            "0",
		"model":           "2201123C",
		"pn":              pn,
		"r":               r,
		"rn":              rn,
		"stErrorNums":     "1",
		"stMethod":        "1",
		"stMode":          "1",
		"stTimesNum":      "1",
		"stTime":          strconv.Itoa(stTime),
		"stSize":          strconv.Itoa(stSize),
		"st_type":         "tb_frslist",
		"with_floor":      "1",
	}

	// ksort + md5 sign
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	// sort keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	var raw strings.Builder
	for _, k := range keys {
		raw.WriteString(k + "=" + params[k])
	}
	params["sign"] = strings.ToUpper(_function.Md5(raw.String() + secret))

	// Build POST body
	var body strings.Builder
	first := true
	for _, k := range keys {
		if !first {
			body.WriteByte('&')
		}
		first = false
		body.WriteString(k + "=" + url.QueryEscape(params[k]))
	}
	body.WriteString("&sign=" + url.QueryEscape(params["sign"]))

	headers := map[string]string{
		"User-Agent": "bdtb for Android 12.41.7.1",
		"Cookie":     "ka=open; BDUSS=" + url.QueryEscape(bduss),
	}

	resp, err := _function.TBFetch("http://c.tieba.baidu.com/c/f/pb/page", http.MethodPost, []byte(body.String()), headers)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = _function.JsonDecode(resp, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RandomInt helper - uses crypto/rand for secure random
func randomInt(min, max int) int {
	n := max - min + 1
	if n <= 0 {
		return min
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	v := int(binary.BigEndian.Uint32(b))
	return min + (v % n)
}

func weltolkGetReplyCount(tid int64, bduss string) (int, error) {
	jsonData, err := weltolkCallTiebaJSONAPI(tid, bduss, "1", "1", "0")
	if err != nil {
		return -1, err
	}
	thread, ok := jsonData["thread"].(map[string]any)
	if !ok {
		return -1, fmt.Errorf("no thread in response")
	}
	replyNum, ok := thread["reply_num"]
	if !ok {
		return -1, fmt.Errorf("no reply_num in thread")
	}
	switch v := replyNum.(type) {
	case float64:
		return int(v), nil
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return -1, err
		}
		return n, nil
	default:
		return -1, fmt.Errorf("unexpected reply_num type")
	}
}

type autoreplyFloorInfo struct {
	ID        any
	AuthorID  any
	Floor     any
	Username  string
	Portrait  string
	Content   string
	SubPosts  []autoreplySubPostInfo
}

type autoreplySubPostInfo struct {
	ID        any
	AuthorID  any
	Username  string
	Portrait  string
	Content   string
}

func weltolkGetLastFloorContent(tid int64, bduss string, limit int) ([]autoreplyFloorInfo, error) {
	jsonData, err := weltolkCallTiebaJSONAPI(tid, bduss, "1", strconv.Itoa(limit), "0")
	if err != nil {
		return nil, err
	}

	postListRaw, ok := jsonData["post_list"]
	if !ok {
		return nil, nil
	}

	postList, ok := postListRaw.([]any)
	if !ok {
		return nil, nil
	}

	var result []autoreplyFloorInfo
	for _, postRaw := range postList {
		post, ok := postRaw.(map[string]any)
		if !ok {
			continue
		}

		authorID := post["author_id"]
		username := ""
		if authorID != nil {
			username = fmt.Sprintf("用户%v", authorID)
		}

		// extract content
		content := ""
		if contentArr, ok := post["content"].([]any); ok {
			for _, cRaw := range contentArr {
				c, ok := cRaw.(map[string]any)
				if !ok {
					continue
				}
				if cType, ok := c["type"].(float64); ok && cType == 0 {
					if text, ok := c["text"].(string); ok {
						content += text
					}
				}
			}
		}

		// parse sub_posts
		var subPosts []autoreplySubPostInfo
		if subPostListRaw, ok := post["sub_post_list"].(map[string]any); ok {
			if subListData, ok := subPostListRaw["sub_post_list"].([]any); ok {
				for _, spRaw := range subListData {
					sp, ok := spRaw.(map[string]any)
					if !ok {
						continue
					}
					spAuthorID := sp["author_id"]
					spUsername := ""
					if spAuthorID != nil {
						spUsername = fmt.Sprintf("用户%v", spAuthorID)
					}
					spPortrait := ""
					if author, ok := sp["author"].(map[string]any); ok {
						if p, ok := author["portrait"].(string); ok {
							spPortrait = p
						}
						if nameShow, ok := author["name_show"].(string); ok && nameShow != "" {
							spUsername = nameShow
						} else if name, ok := author["name"].(string); ok && name != "" {
							spUsername = name
						}
					}
					// extract sub post content
					spContent := ""
					if contentArr, ok := sp["content"].([]any); ok {
						for _, cRaw := range contentArr {
							c, ok := cRaw.(map[string]any)
							if !ok {
								continue
							}
							if cType, ok := c["type"].(float64); ok && cType == 0 {
								if text, ok := c["text"].(string); ok {
									spContent += text
								}
							}
						}
					}
					subPosts = append(subPosts, autoreplySubPostInfo{
						ID:       sp["id"],
						AuthorID: spAuthorID,
						Username: spUsername,
						Portrait: spPortrait,
						Content:  spContent,
					})
				}
			}
		}

		result = append(result, autoreplyFloorInfo{
			ID:       post["id"],
			AuthorID: authorID,
			Floor:    post["floor"],
			Username: username,
			Portrait: "",
			Content:  content,
			SubPosts: subPosts,
		})
	}

	return result, nil
}

// ============================================================
// Add Post API
// ============================================================

type autoreplyAddPostResult struct {
	Success   bool
	ErrorCode int
	ErrorMsg  string
	NeedVcode bool
	RawDebug  string
}

func autoreplyAddPost(bduss, stoken, tbs, fname string, fid int64, tid int64, content, showName, quoteID, replyUID, floorNum, subPostID string) autoreplyAddPostResult {
	// 1. Build protobuf
	protoBinary := autoreplyBuildPostProto(bduss, stoken, tbs, fname, fid, tid, content, showName, quoteID, replyUID, floorNum, subPostID)

	// 2. Build multipart body
	boundary := "-*_r1999"
	var body bytes.Buffer
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"data\"; filename=\"file\"\r\n")
	body.WriteString("\r\n")
	body.Write(protoBinary)
	body.WriteString("\r\n")
	body.WriteString("--" + boundary + "--\r\n")

	// 3. POST
	reqURL := "https://tiebac.baidu.com/c/c/post/add?cmd=309731"
	headers := map[string]string{
		"Content-Type":   "multipart/form-data; boundary=" + boundary,
		"User-Agent":     "tieba/12.35.1.0",
		"x_bd_data_type": "protobuf",
		"Accept-Encoding": "gzip",
		"Connection":      "keep-alive",
		"Cookie":          "BDUSS=" + bduss + "; STOKEN=" + stoken + ";",
	}

	resp, err := _function.TBFetch(reqURL, http.MethodPost, body.Bytes(), headers)
	if err != nil {
		return autoreplyAddPostResult{
			Success:   false,
			ErrorCode: -1,
			ErrorMsg:  "请求失败: " + err.Error(),
			NeedVcode: false,
			RawDebug:  "",
		}
	}

	// 4. Parse protobuf response
	parsed := autoreplyParseResponse(resp)

	success := parsed.ErrorNo == 0
	return autoreplyAddPostResult{
		Success:   success,
		ErrorCode: parsed.ErrorNo,
		ErrorMsg:  parsed.ErrorMsg,
		NeedVcode: parsed.NeedVcode,
		RawDebug:  parsed.RawDebug,
	}
}

// ============================================================
// Action - cron logic
// ============================================================

func (pluginInfo *WeltolkAutoreplyPluginType) Action() {
	if !pluginInfo.PluginInfo.CheckActive() {
		return
	}
	defer pluginInfo.PluginInfo.SetActive(false)

	now := int(time.Now().Unix())
	logTime := time.Now().Format("2006-01-02 15:04:05")

	// High water mark for checkpoint resumption
	highWater, err := strconv.ParseInt(_function.GetOption("weltolk_autoreply_id"), 10, 64)
	if err != nil {
		highWater = 0
	}

	// Query enabled tasks
	var taskList []*model.TcWeltolkAutoreplyTasks
	query := _function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("enabled = 1")
	if highWater > 0 {
		query = query.Where("id >= ?", highWater)
	}
	query.Order("id ASC").Find(&taskList)

	// If no tasks found, reset high water and try from beginning
	if len(taskList) == 0 {
		_function.SetOption("weltolk_autoreply_id", "0")
		_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("enabled = 1").Order("id ASC").Find(&taskList)
	}

	for _, task := range taskList {
		// Initialize context variables
		atUsername := ""
		atPortrait := ""
		quoteID := ""
		replyUID := ""
		floorNum := ""
		subPostID := ""
		keywordMaxSeenPid := int64(0)

		// Skip if reply_content is empty
		if task.ReplyContent == "" {
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"last_status":    "skipped",
				"last_error":     "",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：回复内容为空<br>"),
			})
			continue
		}

		// Step 1: Get BDUSS from tc_baiduid by pid
		pid := task.Pid
		if pid == 0 {
			// Fallback: find first baiduid for this uid
			var baiduid model.TcBaiduid
			_function.GormDB.R.Model(&model.TcBaiduid{}).Where("uid = ?", task.UID).Take(&baiduid)
			if baiduid.ID == 0 {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            0,
					"last_status":    "error",
					"last_error":     "未找到贴吧绑定信息",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：未找到贴吧绑定信息<br>"),
				})
				continue
			}
			pid = int(baiduid.ID)
		}

		cookie := _function.GetCookie(int32(pid), true)
		if !cookie.IsLogin || cookie.Bduss == "" {
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":            pid,
				"last_status":    "error",
				"last_error":     "未获取到BDUSS",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：未获取到BDUSS<br>"),
			})
			continue
		}

		// Step 2: Get reply count
		replyCount, err := weltolkGetReplyCount(task.Tid, cookie.Bduss)
		if err != nil || replyCount < 0 {
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":            pid,
				"last_status":    "error",
				"last_error":     "获取回复数失败",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：获取回复数失败<br>"),
			})
			continue
		}

		// Step 3: Check for new replies
		if task.TriggerMode != "keyword" {
			// new_floor mode
			latestFloors, err := weltolkGetLastFloorContent(task.Tid, cookie.Bduss, 1)
			latestPid := int64(0)
			if err == nil && len(latestFloors) > 0 {
				latestPid = toInt64(latestFloors[0].ID)
			}
			if task.AllowReplied == 0 && latestPid <= task.LastRepliedPid {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            pid,
					"last_status":    "skipped",
					"last_error":     "",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：没有新楼层<br>"),
				})
				continue
			}
			if len(latestFloors) == 0 {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            pid,
					"last_status":    "error",
					"last_error":     "获取楼层内容失败",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：获取楼层内容失败<br>"),
				})
				continue
			}
			latest := latestFloors[0]
			quoteID = toString(latest.ID)
			replyUID = toString(latest.AuthorID)
			floorNum = toString(latest.Floor)
			atUsername = latest.Username
			atPortrait = latest.Portrait
			subPostID = ""
		}

		// keyword mode
		if task.TriggerMode == "keyword" {
			atUsername = ""
			atPortrait = ""
			quoteID = ""
			replyUID = ""
			floorNum = ""
			subPostID = ""

			floors, err := weltolkGetLastFloorContent(task.Tid, cookie.Bduss, 20)
			if err != nil || len(floors) == 0 {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            pid,
					"last_status":    "skipped",
					"last_error":     "获取楼层内容失败",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：获取楼层内容失败<br>"),
				})
				continue
			}

			// Calculate max pid for water mark advancement
			for _, floor := range floors {
				floorID := toInt64(floor.ID)
				if floorID > keywordMaxSeenPid {
					keywordMaxSeenPid = floorID
				}
			}

			// Filter new floors
			var newFloors []autoreplyFloorInfo
			for _, floor := range floors {
				floorID := toInt64(floor.ID)
				if task.AllowReplied == 1 || floorID > task.LastRepliedPid {
					newFloors = append(newFloors, floor)
				}
			}

			if len(newFloors) == 0 {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            pid,
					"last_status":    "skipped",
					"last_error":     "",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：无新楼层<br>"),
				})
				continue
			}

			// Keyword matching
			matched := false
			if strings.TrimSpace(task.MatchKeywords) != "" {
				keywords := strings.Split(task.MatchKeywords, "\n")
				for _, floor := range newFloors {
					floorContent := floor.Content
					if floorContent == "" {
						continue
					}
					for _, kw := range keywords {
						kw = strings.TrimSpace(kw)
						if kw == "" {
							continue
						}
						if strings.Contains(strings.ToLower(floorContent), strings.ToLower(kw)) {
							matched = true
							quoteID = toString(floor.ID)
							floorNum = toString(floor.Floor)

							// Subpost reply mode
							if task.ReplyTarget == "subpost" && len(floor.SubPosts) > 0 {
								subMatched := false
								for _, sp := range floor.SubPosts {
									spContent := sp.Content
									if spContent != "" && strings.Contains(strings.ToLower(spContent), strings.ToLower(kw)) {
										subPostID = toString(sp.ID)
										replyUID = toString(sp.AuthorID)
										atUsername = sp.Username
										atPortrait = sp.Portrait
										subMatched = true
										break
									}
								}
								if !subMatched {
									subPostID = ""
									atUsername = floor.Username
									atPortrait = floor.Portrait
									replyUID = toString(floor.AuthorID)
								}
							} else {
								subPostID = ""
								atUsername = floor.Username
								atPortrait = floor.Portrait
								replyUID = toString(floor.AuthorID)
							}
							break
						}
					}
					if matched {
						break
					}
				}
			}

			if !matched {
				// Keywords not matched - advance water mark
				newLastRepliedPid := task.LastRepliedPid
				if task.AllowReplied != 1 {
					newLastRepliedPid = keywordMaxSeenPid
				}
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":             pid,
					"last_replied_pid": newLastRepliedPid,
					"last_status":     "skipped",
					"last_error":      "",
					"last_check_time": now,
					"log":             gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：关键词未匹配（水位推进至"+strconv.FormatInt(keywordMaxSeenPid, 10)+"）<br>"),
				})
				continue
			}
		}

		// Step 4: Check reply interval
		elapsed := now - task.LastReplyTime
		if task.LastReplyTime > 0 && elapsed < task.ReplyInterval {
			remaining := task.ReplyInterval - elapsed
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":            pid,
				"last_status":    "skipped",
				"last_error":     "",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：回复间隔未到（需等待 "+strconv.Itoa(remaining)+" 秒）<br>"),
			})
			continue
		}

		// Step 5: Check reply probability
		if task.ReplyProbability < 100 {
			randVal := randomInt(1, 100)
			if randVal > task.ReplyProbability {
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":            pid,
					"last_status":    "skipped",
					"last_error":     "",
					"last_check_time": now,
					"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：概率未命中<br>"),
				})
				continue
			}
		}

		// Step 6: Get TBS
		cookieFull := _function.GetCookie(int32(pid))
		if !cookieFull.IsLogin || cookieFull.Tbs == "" {
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":            pid,
				"last_status":    "error",
				"last_error":     "获取TBS失败",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：获取TBS失败<br>"),
			})
			continue
		}
		tbs := cookieFull.Tbs

		// Step 7: Get FID
		fid := _function.GetFid(task.Fname)
		if fid == 0 {
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":            pid,
				"last_status":    "error",
				"last_error":     "获取fid失败",
				"last_check_time": now,
				"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：获取fid失败<br>"),
			})
			continue
		}

		// Step 8: Variable replacement
		floorForReplace := strconv.Itoa(replyCount)
		if task.TriggerMode == "keyword" && floorNum != "" {
			floorForReplace = floorNum
		}
		finalContent := task.ReplyContent
		finalContent = strings.ReplaceAll(finalContent, "{floor}", floorForReplace)
		finalContent = strings.ReplaceAll(finalContent, "{time}", time.Now().Format("2006-01-02 15:04:05"))
		finalContent = strings.ReplaceAll(finalContent, "{date}", time.Now().Format("2006-01-02"))
		finalContent = strings.ReplaceAll(finalContent, "{tid}", strconv.FormatInt(task.Tid, 10))
		finalContent = strings.ReplaceAll(finalContent, "{username}", atUsername)

		// Subpost prefix for keyword+subpost mode
		if task.TriggerMode == "keyword" && task.ReplyTarget == "subpost" && subPostID != "" && atUsername != "" {
			finalContent = "回复 #(reply, " + atPortrait + ", " + atUsername + ") :" + finalContent
		}

		// Step 9: Execute reply
		showName := "贴吧用户"
		result := autoreplyAddPost(cookieFull.Bduss, cookieFull.Stoken, tbs, task.Fname, fid, task.Tid, finalContent, showName, quoteID, replyUID, floorNum, subPostID)

		if result.Success {
			// Success
			newLastRepliedPid := task.LastRepliedPid
			if task.AllowReplied != 1 {
				if task.TriggerMode == "keyword" {
					newLastRepliedPid = keywordMaxSeenPid
				} else {
					newLastRepliedPid = toInt64FromStr(quoteID)
				}
			}
			_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
				"pid":             pid,
				"last_floor":      replyCount,
				"last_replied_pid": newLastRepliedPid,
				"last_reply_time": now,
				"retry_count":     0,
				"last_status":     "ok",
				"last_error":      "",
				"last_check_time": now,
				"log":             gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：操作成功<br>"),
			})
		} else {
			if result.NeedVcode {
				// Vcode - don't increase retry count
				vcodeNewPid := task.LastRepliedPid
				if task.AllowReplied != 1 {
					if task.TriggerMode == "keyword" {
						vcodeNewPid = keywordMaxSeenPid
					} else {
						vcodeNewPid = toInt64FromStr(quoteID)
					}
				}
				_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
					"pid":             pid,
					"last_floor":      replyCount,
					"last_replied_pid": vcodeNewPid,
					"last_status":     "vcode",
					"last_error":      "触发验证码",
					"last_check_time": now,
					"log":             gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", "["+logTime+"] 执行结果：跳过：触发验证码<br>"),
				})
			} else {
				// Error - increase retry count
				newRetry := task.RetryCount + 1
				if newRetry >= 3 {
					_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
						"pid":            pid,
						"last_floor":     replyCount,
						"retry_count":    newRetry,
						"enabled":        0,
						"last_status":    "error",
						"last_error":     fmt.Sprintf("重试次数达上限: [%d] %s", result.ErrorCode, result.ErrorMsg),
						"last_check_time": now,
						"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", fmt.Sprintf("[%s] 执行结果：失败：重试次数达上限，任务已禁用#[%d] %s<br>", logTime, result.ErrorCode, result.ErrorMsg)),
					})
				} else {
					_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", task.ID).Updates(map[string]any{
						"pid":            pid,
						"last_floor":     replyCount,
						"retry_count":    newRetry,
						"last_status":    "error",
						"last_error":     fmt.Sprintf("[错误码 %d]%s", result.ErrorCode, result.ErrorMsg),
						"last_check_time": now,
						"log":            gorm.Expr("CONCAT(COALESCE(`log`, ''), ?)", fmt.Sprintf("[%s] 执行结果：操作失败#[%d] %s<br>", logTime, result.ErrorCode, result.ErrorMsg)),
					})
				}
			}
		}

		// Advance high water mark
		_function.SetOption("weltolk_autoreply_id", strconv.Itoa(int(task.ID)+1))
	}

	// Reset high water mark after full pass
	_function.SetOption("weltolk_autoreply_id", "0")
}

// ============================================================
// Lifecycle methods
// ============================================================

func (pluginInfo *WeltolkAutoreplyPluginType) Install() error {
	for k, v := range pluginInfo.Options {
		_function.SetOption(k, v)
	}
	UpdatePluginInfo(pluginInfo.Name, pluginInfo.Version, false, "")
	return _function.GormDB.W.Migrator().CreateTable(&model.TcWeltolkAutoreplyTasks{})
}

func (pluginInfo *WeltolkAutoreplyPluginType) Delete() error {
	for k := range pluginInfo.Options {
		_function.DeleteOption(k)
	}
	DeletePluginInfo(pluginInfo.Name)
	_function.GormDB.W.Migrator().DropTable(&model.TcWeltolkAutoreplyTasks{})

	// clean user options
	_function.GormDB.W.Where("name = ?", "weltolk_autoreply_open").Delete(&model.TcUsersOption{})
	_function.GormDB.W.Where("name = ?", "weltolk_autoreply_limit").Delete(&model.TcUsersOption{})

	return nil
}

func (pluginInfo *WeltolkAutoreplyPluginType) Upgrade() error {
	return nil
}

func (pluginInfo *WeltolkAutoreplyPluginType) RemoveAccount(_type string, id int32, tx *gorm.DB) error {
	_sql := _function.GormDB.W
	if tx != nil {
		_sql = tx
	}
	return _sql.Where(_type+" = ?", id).Delete(&model.TcWeltolkAutoreplyTasks{}).Error
}

func (pluginInfo *WeltolkAutoreplyPluginType) Report(int32, *gorm.DB) (string, error) {
	return "", nil
}

func (pluginInfo *WeltolkAutoreplyPluginType) Reset(uid, pid, tid int32) error {
	if uid == 0 {
		return errors.New("invalid uid")
	}
	_sql := _function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", uid)
	if pid != 0 {
		_sql = _sql.Where("pid = ?", pid)
	}
	if tid != 0 {
		_sql = _sql.Where("id = ?", tid)
	}
	return _sql.Update("last_reply_time", 0).Error
}

func (pluginInfo *WeltolkAutoreplyPluginType) ExportAccount(uid int32, tx *gorm.DB) (map[string]any, error) {
	if !pluginInfo.GetSwitch() {
		return nil, nil
	}

	if tx == nil {
		tx = _function.GormDB.R
	}

	var taskList []*model.TcWeltolkAutoreplyTasks
	err := tx.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", uid).Find(&taskList).Error
	if err != nil {
		return nil, err
	}

	return map[string]any{
		(&model.TcWeltolkAutoreplyTasks{}).TableName(): taskList,
		"tc_users_options": _function.GetUserOptionBatch(strconv.Itoa(int(uid)), _function.OptionExt{
			Tx:      tx,
			KeyName: "weltolk_autoreply_open",
		}),
	}, nil
}

func (pluginInfo *WeltolkAutoreplyPluginType) ImportAccount(uid int32, pidMap map[int32]int32, data map[string]json.RawMessage, tx *gorm.DB) error {
	if !pluginInfo.GetSwitch() {
		return errors.New("plugin is not enabled")
	}

	if tx == nil {
		tx = _function.GormDB.W
	}

	tableName := (&model.TcWeltolkAutoreplyTasks{}).TableName()

	var data2 []*model.TcWeltolkAutoreplyTasks
	if err := _function.JsonDecode(data[tableName], &data2); err != nil {
		return errors.New("invalid data format")
	}

	var data3 []*model.TcWeltolkAutoreplyTasks

	numLimit, _ := strconv.Atoi(_function.GetOption("weltolk_autoreply_limit"))

	var existsAccountList []*model.TcWeltolkAutoreplyTasks
	_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", uid).Find(&existsAccountList)

	count := len(existsAccountList)
	remain := numLimit - count

	for i := range data2 {
		if newPid, ok := pidMap[int32(data2[i].Pid)]; ok {
			if remain <= 0 {
				break
			}

			var exists bool
			for _, task := range existsAccountList {
				if task.Pid == int(newPid) && task.Tid == data2[i].Tid && task.Fname == data2[i].Fname {
					exists = true
					break
				}
			}

			if !exists {
				data2[i].Pid = int(newPid)
				data2[i].ID = 0
				data2[i].UID = int(uid)

				data3 = append(data3, data2[i])
				remain--
			}
		}
	}

	if len(data3) > 0 {
		return tx.Model(&model.TcWeltolkAutoreplyTasks{}).Create(data3).Error
	}

	return nil
}

// ============================================================
// API Endpoints
// ============================================================

// switch GET
func PluginWeltolkAutoreplyGetSwitch(c echo.Context) error {
	uid := c.Get("uid").(string)
	status := _function.GetUserOption("weltolk_autoreply_open", uid)
	if status == "" {
		status = "0"
		_function.SetUserOption("weltolk_autoreply_open", status, uid)
	}
	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", status != "0", "tbsign"))
}

// switch POST
func PluginWeltolkAutoreplySwitch(c echo.Context) error {
	uid := c.Get("uid").(string)
	status := _function.GetUserOption("weltolk_autoreply_open", uid) != "0"

	err := _function.SetUserOption("weltolk_autoreply_open", !status, uid)
	if err != nil {
		slog.Debug("plugin.weltolk-autoreply.switch", "uid", uid, "current_status", status, "error", err)
		return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "无法启用自动回帖功能", status, "tbsign"))
	}
	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", !status, "tbsign"))
}

// list GET
func PluginWeltolkAutoreplyGetList(c echo.Context) error {
	uid := c.Get("uid").(string)

	var taskList []*model.TcWeltolkAutoreplyTasks
	_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", uid).Order("id ASC").Find(&taskList)

	numLimit, _ := strconv.Atoi(_function.GetOption("weltolk_autoreply_limit"))

	type responseItem struct {
		ID               uint   `json:"id"`
		UID              int    `json:"uid"`
		Pid              int    `json:"pid"`
		Fname            string `json:"fname"`
		Tid              int64  `json:"tid"`
		LastFloor        int    `json:"last_floor"`
		LastRepliedPid   int64  `json:"last_replied_pid"`
		LastReplyTime    int    `json:"last_reply_time"`
		LastStatus       string `json:"last_status"`
		LastError        string `json:"last_error"`
		LastCheckTime    int    `json:"last_check_time"`
		Log              string `json:"log"`
		ReplyContent     string `json:"reply_content"`
		ReplyInterval    int    `json:"reply_interval"`
		ReplyProbability int    `json:"reply_probability"`
		Enabled          int8   `json:"enabled"`
		RetryCount       int    `json:"retry_count"`
		TriggerMode      string `json:"trigger_mode"`
		ReplyTarget      string `json:"reply_target"`
		AllowReplied     int8   `json:"allow_replied"`
		MatchKeywords    string `json:"match_keywords"`
	}

	var responseList []responseItem
	for _, v := range taskList {
		responseList = append(responseList, responseItem{
			ID:               v.ID,
			UID:              v.UID,
			Pid:              v.Pid,
			Fname:            v.Fname,
			Tid:              v.Tid,
			LastFloor:        v.LastFloor,
			LastRepliedPid:   v.LastRepliedPid,
			LastReplyTime:    v.LastReplyTime,
			LastStatus:       v.LastStatus,
			LastError:        v.LastError,
			LastCheckTime:    v.LastCheckTime,
			Log:              v.Log,
			ReplyContent:     v.ReplyContent,
			ReplyInterval:    v.ReplyInterval,
			ReplyProbability: v.ReplyProbability,
			Enabled:          v.Enabled,
			RetryCount:       v.RetryCount,
			TriggerMode:      v.TriggerMode,
			ReplyTarget:      v.ReplyTarget,
			AllowReplied:     v.AllowReplied,
			MatchKeywords:    v.MatchKeywords,
		})
	}

	type listResponse struct {
		Count int64          `json:"count"`
		Limit int64          `json:"limit"`
		List  []responseItem `json:"list"`
	}

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", listResponse{
		Count: int64(len(responseList)),
		Limit: int64(numLimit),
		List:  responseList,
	}, "tbsign"))
}

// list PATCH - Add task
func PluginWeltolkAutoreplyAddTask(c echo.Context) error {
	uid := c.Get("uid").(string)
	numUID, _ := strconv.ParseInt(uid, 10, 64)

	pidStr := c.FormValue("pid")
	fname := c.FormValue("fname")
	tidStr := c.FormValue("tid")
	replyContent := c.FormValue("reply_content")
	replyIntervalStr := c.FormValue("reply_interval")
	replyProbabilityStr := c.FormValue("reply_probability")
	triggerMode := c.FormValue("trigger_mode")
	matchKeywords := c.FormValue("match_keywords")
	replyTarget := c.FormValue("reply_target")
	allowRepliedStr := c.FormValue("allow_replied")
	enabledStr := c.FormValue("enabled")

	// Parse pid - use the pid from frontend directly (BUG3 fix)
	numPid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "无效 pid", _function.EchoEmptyObject, "tbsign"))
	}

	// Validate pid belongs to user
	var accountInfo model.TcBaiduid
	_function.GormDB.R.Model(&model.TcBaiduid{}).Where("id = ? AND uid = ?", numPid, uid).Take(&accountInfo)
	if accountInfo.Portrait == "" {
		return c.JSON(http.StatusNotFound, _function.ApiTemplate(404, "无效 pid", _function.EchoEmptyObject, "tbsign"))
	}

	// Parse tid
	numTid, err := strconv.ParseInt(tidStr, 10, 64)
	if err != nil || numTid == 0 {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "无效 tid", _function.EchoEmptyObject, "tbsign"))
	}

	if fname == "" || replyContent == "" {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "请填写所有必填字段", _function.EchoEmptyObject, "tbsign"))
	}

	// Parse optional fields
	replyInterval := 300
	if v, err := strconv.Atoi(replyIntervalStr); err == nil && v > 0 {
		replyInterval = v
	}
	replyProbability := 100
	if v, err := strconv.Atoi(replyProbabilityStr); err == nil && v > 0 && v <= 100 {
		replyProbability = v
	}
	if triggerMode == "" {
		triggerMode = "new_floor"
	}
	if replyTarget == "" {
		replyTarget = "floor"
	}
	allowReplied := int8(0)
	if allowRepliedStr == "1" {
		allowReplied = 1
	}
	enabled := int8(1)
	if enabledStr == "0" {
		enabled = 0
	}

	// Check limit
	personalLimit := 0
	personalLimitStr := _function.GetUserOption("weltolk_autoreply_limit", uid)
	if personalLimitStr != "" {
		if v, err := strconv.Atoi(personalLimitStr); err == nil && v > 0 {
			personalLimit = v
		}
	}
	globalLimit, _ := strconv.Atoi(_function.GetOption("weltolk_autoreply_limit"))
	limit := globalLimit
	if personalLimit > 0 {
		limit = personalLimit
	}
	if limit <= 0 {
		limit = 5
	}

	var count int64
	_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", uid).Count(&count)
	if int(count) >= limit {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, fmt.Sprintf("已达到最大任务数限制（%d 条）", limit), _function.EchoEmptyObject, "tbsign"))
	}

	// Create task
	newTask := &model.TcWeltolkAutoreplyTasks{
		UID:              int(numUID),
		Pid:              int(numPid),
		Fname:            fname,
		Tid:              numTid,
		ReplyContent:     replyContent,
		ReplyInterval:    replyInterval,
		ReplyProbability: replyProbability,
		Enabled:          enabled,
		TriggerMode:      triggerMode,
		ReplyTarget:      replyTarget,
		AllowReplied:     allowReplied,
		MatchKeywords:    matchKeywords,
	}

	err = _function.GormDB.W.Create(newTask).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "创建任务失败", _function.EchoEmptyObject, "tbsign"))
	}

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", newTask, "tbsign"))
}

// list PUT /:id - Edit task
func PluginWeltolkAutoreplyEditTask(c echo.Context) error {
	uid := c.Get("uid").(string)
	id := c.Param("id")

	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "无效 id", _function.EchoEmptyObject, "tbsign"))
	}

	// Check ownership
	var existingTask model.TcWeltolkAutoreplyTasks
	_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ? AND uid = ?", numID, uid).Take(&existingTask)
	if existingTask.ID == 0 {
		return c.JSON(http.StatusNotFound, _function.ApiTemplate(404, "任务不存在", _function.EchoEmptyObject, "tbsign"))
	}

	// Parse fields
	pidStr := c.FormValue("pid")
	fname := c.FormValue("fname")
	tidStr := c.FormValue("tid")
	replyContent := c.FormValue("reply_content")
	replyIntervalStr := c.FormValue("reply_interval")
	replyProbabilityStr := c.FormValue("reply_probability")
	triggerMode := c.FormValue("trigger_mode")
	matchKeywords := c.FormValue("match_keywords")
	replyTarget := c.FormValue("reply_target")
	allowRepliedStr := c.FormValue("allow_replied")
	enabledStr := c.FormValue("enabled")

	updates := map[string]any{}

	if pidStr != "" {
		numPid, err := strconv.ParseInt(pidStr, 10, 64)
		if err == nil {
			// Validate pid belongs to user
			var accountInfo model.TcBaiduid
			_function.GormDB.R.Model(&model.TcBaiduid{}).Where("id = ? AND uid = ?", numPid, uid).Take(&accountInfo)
			if accountInfo.Portrait != "" {
				updates["pid"] = int(numPid)
			}
		}
	}

	if fname != "" {
		updates["fname"] = fname
	}
	if tidStr != "" {
		numTid, err := strconv.ParseInt(tidStr, 10, 64)
		if err == nil {
			updates["tid"] = numTid
		}
	}
	if replyContent != "" {
		updates["reply_content"] = replyContent
	}
	if replyIntervalStr != "" {
		if v, err := strconv.Atoi(replyIntervalStr); err == nil && v > 0 {
			updates["reply_interval"] = v
		}
	}
	if replyProbabilityStr != "" {
		if v, err := strconv.Atoi(replyProbabilityStr); err == nil && v > 0 && v <= 100 {
			updates["reply_probability"] = v
		}
	}
	if triggerMode != "" {
		updates["trigger_mode"] = triggerMode
	}
	if matchKeywords != "" || triggerMode == "keyword" {
		updates["match_keywords"] = matchKeywords
	}
	if replyTarget != "" {
		updates["reply_target"] = replyTarget
	}
	if allowRepliedStr != "" {
		updates["allow_replied"] = int8(0)
		if allowRepliedStr == "1" {
			updates["allow_replied"] = int8(1)
		}
	}
	if enabledStr != "" {
		updates["enabled"] = int8(0)
		if enabledStr == "1" {
			updates["enabled"] = int8(1)
		}
	}

	if len(updates) > 0 {
		_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ? AND uid = ?", numID, uid).Updates(updates)
	}

	// Return updated task
	_function.GormDB.R.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ?", numID).Take(&existingTask)
	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", existingTask, "tbsign"))
}

// list DELETE /:id
func PluginWeltolkAutoreplyDelTask(c echo.Context) error {
	uid := c.Get("uid").(string)
	id := c.Param("id")

	numUID, _ := strconv.ParseInt(uid, 10, 64)
	numID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "无效 id", map[string]any{
			"success": false,
			"id":      id,
		}, "tbsign"))
	}

	_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("id = ? AND uid = ?", numID, numUID).Delete(&model.TcWeltolkAutoreplyTasks{})

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", map[string]any{
		"success": true,
		"id":      id,
	}, "tbsign"))
}

// list /empty POST
func PluginWeltolkAutoreplyDelAllTasks(c echo.Context) error {
	uid := c.Get("uid").(string)
	numUID, _ := strconv.ParseInt(uid, 10, 64)

	_function.GormDB.W.Model(&model.TcWeltolkAutoreplyTasks{}).Where("uid = ?", numUID).Delete(&model.TcWeltolkAutoreplyTasks{})

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", true, "tbsign"))
}

// test POST
func PluginWeltolkAutoreplyTest(c echo.Context) error {
	uid := c.Get("uid").(string)

	fname := c.FormValue("fname")
	tidStr := c.FormValue("tid")
	replyContent := c.FormValue("reply_content")
	pidStr := c.FormValue("pid")
	triggerMode := c.FormValue("trigger_mode")
	matchKeywords := c.FormValue("match_keywords")
	replyTarget := c.FormValue("reply_target")

	numPid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "无效 pid", _function.EchoEmptyObject, "tbsign"))
	}

	// Validate pid belongs to user
	var accountInfo model.TcBaiduid
	_function.GormDB.R.Model(&model.TcBaiduid{}).Where("id = ? AND uid = ?", numPid, uid).Take(&accountInfo)
	if accountInfo.Portrait == "" {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "越权操作：该百度账号不属于您", _function.EchoEmptyObject, "tbsign"))
	}

	numTid, err := strconv.ParseInt(tidStr, 10, 64)
	if err != nil || numTid == 0 {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "请填写帖子ID", _function.EchoEmptyObject, "tbsign"))
	}
	if fname == "" || replyContent == "" {
		return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "请填写所有必填字段", _function.EchoEmptyObject, "tbsign"))
	}

	cookie := _function.GetCookie(int32(numPid))
	if !cookie.IsLogin || cookie.Bduss == "" {
		return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "无法获取BDUSS", _function.EchoEmptyObject, "tbsign"))
	}

	tbs := cookie.Tbs
	fid := _function.GetFid(fname)

	// Get floor info
	quoteID := ""
	replyUID := ""
	floorNum := ""
	subPostID := ""
	atUsername := ""
	atPortrait := ""
	testInfo := ""
	skipTest := false

	if triggerMode == "keyword" {
		if strings.TrimSpace(matchKeywords) == "" {
			// No keywords set, will post as topic reply
		} else {
			floors, err := weltolkGetLastFloorContent(numTid, cookie.Bduss, 20)
			if err != nil || len(floors) == 0 {
				return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "获取楼层内容失败，无法测试关键词匹配", _function.EchoEmptyObject, "tbsign"))
			}

			keywords := strings.Split(matchKeywords, "\n")
			matched := false
			for _, floor := range floors {
				floorContent := floor.Content
				if floorContent == "" {
					continue
				}
				for _, kw := range keywords {
					kw = strings.TrimSpace(kw)
					if kw == "" {
						continue
					}
					if strings.Contains(strings.ToLower(floorContent), strings.ToLower(kw)) {
						matched = true
						quoteID = toString(floor.ID)
						floorNum = toString(floor.Floor)
						atUsername = floor.Username
						atPortrait = floor.Portrait
						replyUID = toString(floor.AuthorID)

						// Subpost mode
						if replyTarget == "subpost" && len(floor.SubPosts) > 0 {
							subMatched := false
							for _, sp := range floor.SubPosts {
								spContent := sp.Content
								if spContent != "" && strings.Contains(strings.ToLower(spContent), strings.ToLower(kw)) {
									subPostID = toString(sp.ID)
									replyUID = toString(sp.AuthorID)
									atUsername = sp.Username
									atPortrait = sp.Portrait
									subMatched = true
									break
								}
							}
							if !subMatched {
								subPostID = ""
								atUsername = floor.Username
								atPortrait = floor.Portrait
								replyUID = toString(floor.AuthorID)
							}
						}
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				return c.JSON(http.StatusOK, _function.ApiTemplate(200, "关键词未匹配任何楼层", map[string]any{
					"matched": false,
					"info":    "最新 " + strconv.Itoa(len(floors)) + " 个楼层中未找到任何匹配关键词的内容",
				}, "tbsign"))
			}
			testInfo = fmt.Sprintf("（关键词匹配 @#%s %s）", floorNum, atUsername)
		}
	} else {
		// new_floor mode
		latestFloors, err := weltolkGetLastFloorContent(numTid, cookie.Bduss, 1)
		if err == nil && len(latestFloors) > 0 {
			latest := latestFloors[0]
			quoteID = toString(latest.ID)
			replyUID = toString(latest.AuthorID)
			floorNum = toString(latest.Floor)
			atUsername = latest.Username
			atPortrait = latest.Portrait
			testInfo = fmt.Sprintf("（回复 @#%s %s）", floorNum, atUsername)
		}
	}

	// Variable replacement
	floorForReplace := floorNum
	if floorForReplace == "" {
		floorForReplace = "测试"
	}
	content := replyContent
	content = strings.ReplaceAll(content, "{floor}", floorForReplace)
	content = strings.ReplaceAll(content, "{time}", time.Now().Format("2006-01-02 15:04:05"))
	content = strings.ReplaceAll(content, "{date}", time.Now().Format("2006-01-02"))
	content = strings.ReplaceAll(content, "{tid}", strconv.FormatInt(numTid, 10))
	content = strings.ReplaceAll(content, "{username}", atUsername)

	// Subpost prefix
	if triggerMode == "keyword" && replyTarget == "subpost" && subPostID != "" && atUsername != "" {
		content = "回复 #(reply, " + atPortrait + ", " + atUsername + ") :" + content
	}

	if skipTest {
		return c.JSON(http.StatusOK, _function.ApiTemplate(200, "跳过测试", map[string]any{
			"skipped": true,
		}, "tbsign"))
	}

	result := autoreplyAddPost(cookie.Bduss, cookie.Stoken, tbs, fname, fid, numTid, content, "贴吧用户", quoteID, replyUID, floorNum, subPostID)

	type testResult struct {
		Success   bool   `json:"success"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		NeedVcode bool   `json:"need_vcode"`
		Info      string `json:"info,omitempty"`
		RawDebug  string `json:"raw_debug,omitempty"`
	}

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", testResult{
		Success:   result.Success,
		ErrorCode: result.ErrorCode,
		ErrorMsg:  result.ErrorMsg,
		NeedVcode: result.NeedVcode,
		Info:      testInfo,
		RawDebug:  result.RawDebug,
	}, "tbsign"))
}

// settings GET
func PluginWeltolkAutoreplyGetSettings(c echo.Context) error {
	uid := c.Get("uid").(string)

	globalLimit, _ := strconv.Atoi(_function.GetOption("weltolk_autoreply_limit"))
	personalLimit := 0
	personalLimitStr := _function.GetUserOption("weltolk_autoreply_limit", uid)
	if personalLimitStr != "" {
		if v, err := strconv.Atoi(personalLimitStr); err == nil && v > 0 {
			personalLimit = v
		}
	}

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", map[string]any{
		"global_limit":   globalLimit,
		"personal_limit": personalLimit,
	}, "tbsign"))
}

// settings PUT
func PluginWeltolkAutoreplySetSettings(c echo.Context) error {
	uid := c.Get("uid").(string)
	personalLimitStr := c.FormValue("personal_limit")

	personalLimit := 0
	if personalLimitStr != "" {
		v, err := strconv.Atoi(personalLimitStr)
		if err != nil || v < 0 {
			return c.JSON(http.StatusForbidden, _function.ApiTemplate(403, "无效的限制值", _function.EchoEmptyObject, "tbsign"))
		}
		personalLimit = v
	}

	if personalLimit == 0 {
		// Clear personal override, use global default
		_function.DeleteUserOption("weltolk_autoreply_limit", uid)
	} else {
		err := _function.SetUserOption("weltolk_autoreply_limit", personalLimit, uid)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, _function.ApiTemplate(500, "无法更新设置", _function.EchoEmptyObject, "tbsign"))
		}
	}

	globalLimit, _ := strconv.Atoi(_function.GetOption("weltolk_autoreply_limit"))

	return c.JSON(http.StatusOK, _function.ApiTemplate(200, "OK", map[string]any{
		"global_limit":   globalLimit,
		"personal_limit": personalLimit,
	}, "tbsign"))
}

// ============================================================
// Helper functions
// ============================================================

func toInt64(v any) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case string:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0
		}
		return n
	case int:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt64FromStr(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
