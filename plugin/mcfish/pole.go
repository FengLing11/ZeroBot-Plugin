// Package mcfish 钓鱼模拟器
package mcfish

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/FloatTech/zbputils/ctxext"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
)

func init() {
	engine.OnRegex(`^装备(`+strings.Join(poleList, "|")+`)$`, getdb).SetBlock(true).Limit(ctxext.LimitByUser).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		equipInfo, err := dbdata.getUserEquip(uid)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.1]:", err))
			return
		}
		thingName := ctx.State["regex_matched"].([]string)[1]
		articles, err := dbdata.getUserThingInfo(uid, thingName)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.2]:", err))
			return
		}
		if len(articles) == 0 {
			ctx.SendChain(message.Text("你的背包不存在该物品"))
			return
		}
		poles := make([]equip, 0, len(articles))
		if thingName != "美西螈" {
			for _, info := range articles {
				poleInfo := strings.Split(info.Other, "/")
				durable, _ := strconv.Atoi(poleInfo[0])
				maintenance, _ := strconv.Atoi(poleInfo[1])
				induceLevel, _ := strconv.Atoi(poleInfo[2])
				favorLevel, _ := strconv.Atoi(poleInfo[3])
				poles = append(poles, equip{
					ID:          uid,
					Equip:       info.Name,
					Durable:     durable,
					Maintenance: maintenance,
					Induce:      induceLevel,
					Favor:       favorLevel,
				})
			}
		} else {
			poles = append(poles, equip{
				ID:      uid,
				Equip:   thingName,
				Durable: 999,
			})
		}
		check := false
		index := 0
		if len(poles) > 1 {
			msg := make(message.Message, 0, 3+len(articles))
			msg = append(msg, message.Reply(ctx.Event.MessageID), message.Text("找到以下鱼竿:\n"))
			for i, info := range poles {
				msg = append(msg, message.Text("[", i, "] ", info.Equip, " : 耐", info.Durable, "/修", info.Maintenance,
					"/诱", enchantLevel[info.Induce], "/眷顾", enchantLevel[info.Favor], "\n"))
			}
			msg = append(msg, message.Text("————————\n"))
			msg = append(msg, message.Text("- 输入对应序号进行装备\n"))
			msg = append(msg, message.Text("- 输入“取消”终止本次操作\n"))
			msg = append(msg, message.Text("- 鱼竿数量请使用钓鱼背包查看"))
			ctx.Send(msg)
			// 等待用户下一步选择
			recv, cancel := zero.NewFutureEvent("message", 999, false, zero.RegexRule(`^(取消|\d+)$`), zero.CheckUser(ctx.Event.UserID)).Repeat()
			defer cancel()
			for {
				select {
				case <-time.After(time.Second * 120):
					ctx.Send(
						message.ReplyWithMessage(ctx.Event.MessageID,
							message.Text("等待超时,取消装备"),
						),
					)
					return
				case e := <-recv:
					nextcmd := e.Event.Message.String()
					if nextcmd == "取消" {
						ctx.Send(
							message.ReplyWithMessage(ctx.Event.MessageID,
								message.Text("已取消装备"),
							),
						)
						return
					}
					index, err = strconv.Atoi(nextcmd)
					if err != nil || index > len(articles)-1 {
						ctx.SendChain(message.At(ctx.Event.UserID), message.Text("请输入正确的序号"))
						continue
					}
					check = true
				}
				if check {
					break
				}
			}
		}
		newEquipInfo := poles[index]
		packEquip := articles[index]
		packEquip.Number--
		err = dbdata.updateUserThingInfo(uid, packEquip)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.3]:", err))
			return
		}
		err = dbdata.updateUserEquip(newEquipInfo)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.3.1]:", err))
			return
		}
		oldthing := article{}
		if equipInfo != (equip{}) && equipInfo.Equip != "美西螈" {
			oldthing = article{
				Duration: time.Now().Unix(),
				Type:     "pole",
				Name:     equipInfo.Equip,
				Number:   1,
				Other:    strconv.Itoa(equipInfo.Durable) + "/" + strconv.Itoa(equipInfo.Maintenance) + "/" + strconv.Itoa(equipInfo.Induce) + "/" + strconv.Itoa(equipInfo.Favor),
			}
		} else if equipInfo.Equip == "美西螈" {
			articles, err = dbdata.getUserThingInfo(uid, "美西螈")
			if err != nil {
				ctx.SendChain(message.Text("[ERROR at pole.go.3.2]:", err))
				return
			}
			if len(articles) == 0 {
				oldthing = article{
					Duration: time.Now().Unix(),
					Type:     "pole",
					Name:     equipInfo.Equip,
					Number:   1,
				}
			} else {
				oldthing = articles[0]
				oldthing.Number++
			}
		}
		err = dbdata.updateUserThingInfo(uid, oldthing)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.4]:", err))
			return
		}
		ctx.Send(
			message.ReplyWithMessage(ctx.Event.MessageID,
				message.Text("装备成功"),
			),
		)
	})
	engine.OnFullMatchGroup([]string{"修复鱼竿", "维修鱼竿"}, getdb).SetBlock(true).Limit(ctxext.LimitByUser).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		equipInfo, err := dbdata.getUserEquip(uid)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.5]:", err))
			return
		}
		if equipInfo.Equip == "" || equipInfo.Equip == "美西螈" {
			ctx.SendChain(message.Text("仅能修复装备中的鱼竿"))
			return
		}
		if equipInfo.Maintenance >= 10 {
			ctx.SendChain(message.Text("装备的鱼竿已经达到修复上限"))
			return
		}
		articles, err := dbdata.getUserThingInfo(uid, equipInfo.Equip)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.6]:", err))
			return
		}
		if len(articles) == 0 {
			ctx.SendChain(message.Text("你的背包不存在相同鱼竿进行修复"))
			return
		}
		poles := make([]equip, 0, len(articles))
		for _, info := range articles {
			poleInfo := strings.Split(info.Other, "/")
			durable, _ := strconv.Atoi(poleInfo[0])
			maintenance, _ := strconv.Atoi(poleInfo[1])
			induceLevel, _ := strconv.Atoi(poleInfo[2])
			favorLevel, _ := strconv.Atoi(poleInfo[3])
			poles = append(poles, equip{
				ID:          uid,
				Equip:       info.Name,
				Durable:     durable,
				Maintenance: maintenance,
				Induce:      induceLevel,
				Favor:       favorLevel,
			})
		}
		index := 0
		check := false
		if len(articles) > 1 {
			msg := make(message.Message, 0, 3+len(articles))
			msg = append(msg, message.Text("找到以下鱼竿:\n"))
			for i, info := range poles {
				msg = append(msg, message.Text("[", i, "] ", info.Equip, " : 耐", info.Durable, "/修", info.Maintenance,
					"/诱", enchantLevel[info.Induce], "/眷顾", enchantLevel[info.Favor], "\n"))
			}
			msg = append(msg, message.Text("————————\n输入对应序号进行修复,或回复“取消”取消"))
			ctx.Send(message.ReplyWithMessage(ctx.Event.MessageID, msg...))
			// 等待用户下一步选择
			recv, cancel := zero.NewFutureEvent("message", 999, false, zero.RegexRule(`^(取消|\d+)$`), zero.CheckUser(ctx.Event.UserID)).Repeat()
			defer cancel()
			for {
				select {
				case <-time.After(time.Second * 120):
					ctx.Send(
						message.ReplyWithMessage(ctx.Event.MessageID,
							message.Text("等待超时,取消修复"),
						),
					)
					return
				case e := <-recv:
					nextcmd := e.Event.Message.String()
					if nextcmd == "取消" {
						ctx.Send(
							message.ReplyWithMessage(ctx.Event.MessageID,
								message.Text("已取消修复"),
							),
						)
						return
					}
					index, err = strconv.Atoi(nextcmd)
					if err != nil || index > len(articles)-1 {
						ctx.SendChain(message.At(ctx.Event.UserID), message.Text("请输入正确的序号"))
						continue
					}
					check = true
				}
				if check {
					break
				}
			}
		}
		newEquipInfo := poles[index]
		number, err := dbdata.getNumberFor(uid, "竿")
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at fish.go.5.1]:", err))
			return
		}
		if number <= 10 {
			number = 8
		} else {
			number = 10
		}
		equipInfo.Durable += newEquipInfo.Durable * number / 10
		if equipInfo.Durable > durationList[equipInfo.Equip] || equipInfo.Equip == "三叉戟" {
			equipInfo.Durable = durationList[equipInfo.Equip]
		}
		msg := ""
		if newEquipInfo.Induce != 0 && rand.Intn(100) < 50 {
			equipInfo.Induce += newEquipInfo.Induce
			if equipInfo.Induce > 3 {
				equipInfo.Induce = 3
			}
			msg += ",诱钓等级提升至" + enchantLevel[equipInfo.Induce]
		}
		if newEquipInfo.Favor != 0 && rand.Intn(100) < 50 {
			equipInfo.Favor += newEquipInfo.Favor
			if equipInfo.Favor > 3 {
				equipInfo.Favor = 3
			}
			msg += ",海之眷顾等级提升至" + enchantLevel[equipInfo.Favor]
		}
		thingInfo := articles[index]
		thingInfo.Number = 0
		err = dbdata.updateUserThingInfo(uid, thingInfo)
		if err == nil {
			equipInfo.Maintenance++
			err = dbdata.updateUserEquip(equipInfo)
		}
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.7]:", err))
			return
		}
		ctx.Send(
			message.ReplyWithMessage(ctx.Event.MessageID,
				message.Text("鱼竿修复成功,耐久提高至", equipInfo.Durable, msg),
			),
		)
	})
	engine.OnRegex(`^附魔(诱钓|海之眷顾)$`, getdb).SetBlock(true).Limit(ctxext.LimitByUser).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		equipInfo, err := dbdata.getUserEquip(uid)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.7]:", err))
			return
		}
		if equipInfo.Equip == "" || equipInfo.Equip == "美西螈" {
			ctx.SendChain(message.Text("仅可对装备中的进行附魔"))
			return
		}
		book := ctx.State["regex_matched"].([]string)[1]
		books, err := dbdata.getUserThingInfo(uid, book)
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.8]:", err))
			return
		}
		if len(books) == 0 {
			ctx.SendChain(message.Text("你的背包不存在", book, "进行附魔"))
			return
		}
		bookInfo := books[0]
		bookInfo.Number--
		err = dbdata.updateUserThingInfo(uid, bookInfo)
		number := 0
		if err == nil {
			if rand.Intn(100) > 50 {
				ctx.SendChain(message.Text("附魔失败了"))
				return
			}
			switch book {
			case "诱钓":
				equipInfo.Induce++
				if equipInfo.Induce > 3 {
					ctx.SendChain(message.Text("诱钓等级已达到上限，你浪费了一本附魔书"))
					return
				}
				number = equipInfo.Induce
			case "海之眷顾":
				equipInfo.Favor++
				if equipInfo.Favor > 3 {
					ctx.SendChain(message.Text("海之眷顾等级已达到上限，你浪费了一本附魔书"))
					return
				}
				number = equipInfo.Favor
			default:
				ctx.SendChain(message.Text("附魔失败了"))
				return
			}
			err = dbdata.updateUserEquip(equipInfo)
		}
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.9]:", err))
			return
		}
		ctx.SendChain(message.Text("附魔成功,", book, "等级提高至", enchantLevel[number]))
	})
	engine.OnRegex(`^合成(.+竿|三叉戟)$`, getdb).SetBlock(true).Limit(ctxext.LimitByUser).Handle(func(ctx *zero.Ctx) {
		uid := ctx.Event.UserID
		thingList := []string{"木竿", "铁竿", "金竿", "钻石竿", "下界合金竿", "三叉戟"}
		thingName := ctx.State["regex_matched"].([]string)[1]
		indexOfMaterial := -1
		for i, name := range thingList {
			if thingName == name {
				indexOfMaterial = (i - 1)
				break
			}
		}
		if indexOfMaterial < 0 {
			return
		}
		articles, err := dbdata.getUserThingInfo(uid, thingList[indexOfMaterial])
		if err != nil {
			ctx.SendChain(message.Text("[ERROR at pole.go.10]:", err))
			return
		}
		maxCount := len(articles)
		if maxCount < 3 {
			ctx.SendChain(message.Reply(ctx.Event.MessageID), message.Text("你的合成材料不足"))
			return
		}
		poles := make([]equip, 0, maxCount)
		for _, info := range articles {
			poleInfo := strings.Split(info.Other, "/")
			durable, _ := strconv.Atoi(poleInfo[0])
			maintenance, _ := strconv.Atoi(poleInfo[1])
			induceLevel, _ := strconv.Atoi(poleInfo[2])
			favorLevel, _ := strconv.Atoi(poleInfo[3])
			poles = append(poles, equip{
				ID:          uid,
				Equip:       info.Name,
				Durable:     durable,
				Maintenance: maintenance,
				Induce:      induceLevel,
				Favor:       favorLevel,
			})
		}
		list := []int{0, 1, 2}
		check := false
		if len(articles) > 3 {
			msg := make(message.Message, 0, 3+len(articles))
			msg = append(msg, message.Text("找到以下鱼竿:\n"))
			for i, info := range poles {
				msg = append(msg, message.Text("[", i, "] ", info.Equip, " : 耐", info.Durable, "/修", info.Maintenance,
					"/诱", enchantLevel[info.Induce], "/眷顾", enchantLevel[info.Favor], "\n"))
			}
			msg = append(msg, message.Text("————————\n"))
			msg = append(msg, message.Text("- 输入3个序号进行合成(用空格分割)\n"))
			msg = append(msg, message.Text("- 输入“取消”，终止本次合成\n"))
			msg = append(msg, message.Text("- 输入“梭哈“，合成所有鱼竿"))
			ctx.Send(message.ReplyWithMessage(ctx.Event.MessageID, msg...))
			// 等待用户下一步选择
			recv, cancel := zero.NewFutureEvent("message", 999, false, zero.RegexRule(`^(梭哈|取消|\d+ \d+ \d+)$`), zero.CheckUser(ctx.Event.UserID)).Repeat()
			defer cancel()
			for {
				select {
				case <-time.After(time.Second * 120):
					ctx.Send(
						message.ReplyWithMessage(ctx.Event.MessageID,
							message.Text("等待超时,取消合成"),
						),
					)
					return
				case e := <-recv:
					nextcmd := e.Event.Message.String()
					if nextcmd == "取消" {
						ctx.Send(
							message.ReplyWithMessage(ctx.Event.MessageID,
								message.Text("已取消合成"),
							),
						)
						return
					}
					if nextcmd == "梭哈" {
						// len(list)取3的倍数，表示能够用于合成鱼竿的最大数量，note：此处未对article.Number>1的情况做处理
						for i := 3; i < (len(articles)/3)*3; i++ {
							list = append(list, i)
						}
						check = true
						break
					}
					chooseList := strings.Split(nextcmd, " ")
					first, err := strconv.Atoi(chooseList[0])
					if err != nil {
						ctx.SendChain(message.Text("[ERROR at pole.go.11.1]:", err))
						return
					}
					second, err := strconv.Atoi(chooseList[1])
					if err != nil {
						ctx.SendChain(message.Text("[ERROR at pole.go.11.2]:", err))
						return
					}
					third, err := strconv.Atoi(chooseList[2])
					if err != nil {
						ctx.SendChain(message.Text("[ERROR at pole.go.11.3]:", err))
						return
					}
					list = []int{first, second, third}
					if first == second || first == third || second == third {
						ctx.SendChain(message.At(ctx.Event.UserID), message.Text("[0]请输入正确的序号\n", list))
						continue
					}
					if first >= maxCount || second >= maxCount || third >= maxCount {
						ctx.SendChain(message.At(ctx.Event.UserID), message.Text("[", maxCount, "]请输入正确的序号\n", list))
						continue
					}
					check = true
				}
				if check {
					break
				}
			}
		}
    groupCount := len(list) / 3
    successCount := 0
    failCount := 0
    successAttributes := []string{} 

    for g := 0; g < groupCount; g++ {
        idx1 := list[g*3]
        idx2 := list[g*3+1]
        idx3 := list[g*3+2]

        thingInfo1 := articles[idx1]
        thingInfo2 := articles[idx2]
        thingInfo3 := articles[idx3]

        thingInfo1.Number = 0
        thingInfo2.Number = 0
        thingInfo3.Number = 0

        err = dbdata.updateUserThingInfo(uid, thingInfo1)
        if err != nil {
            ctx.SendChain(message.Text("[ERROR at pole.go.12]:", err))
            return
        }
        err = dbdata.updateUserThingInfo(uid, thingInfo2)
        if err != nil {
            ctx.SendChain(message.Text("[ERROR at pole.go.12]:", err))
            return
        }
        err = dbdata.updateUserThingInfo(uid, thingInfo3)
        if err != nil {
            ctx.SendChain(message.Text("[ERROR at pole.go.12]:", err))
            return
        }

        if rand.Intn(100) >= 90 {
            failCount++
            continue
        }

        favorLevel := (poles[idx1].Favor + poles[idx2].Favor + poles[idx3].Favor) / 3
        induceLevel := (poles[idx1].Induce + poles[idx2].Induce + poles[idx3].Induce) / 3
        attribute := strconv.Itoa(durationList[thingName]) + "/0/" + strconv.Itoa(induceLevel) + "/" + strconv.Itoa(favorLevel)

        newthing := article{
            Duration: time.Now().Unix() + int64(g*10), 
            Type:     "pole",
            Name:     thingName,
            Number:   1,
            Other:    attribute,
        }
        err = dbdata.updateUserThingInfo(uid, newthing)
        if err != nil {
            ctx.SendChain(message.Text("[ERROR at pole.go.12]:", err))
            return
        }
        successCount++
        successAttributes = append(successAttributes, attribute)
    }

    resultMsg := ""
    if successCount > 0 {
        resultMsg += "成功合成 " + strconv.Itoa(successCount) + " 个" + thingName + "\n"
        if len(successAttributes) <= 5 {
            for _, attr := range successAttributes {
                resultMsg += "属性: " + attr + "\n"
            }
        } else {
            resultMsg += "属性示例: " + successAttributes[0] + " 等\n"
        }
    }
    if failCount > 0 {
        resultMsg += "失败 " + strconv.Itoa(failCount) + " 次，材料已销毁。"
    }
    if resultMsg == "" {
        resultMsg = "没有进行任何合成？"
    }

    ctx.Send(
        message.ReplyWithMessage(ctx.Event.MessageID,
            message.Text(resultMsg),
        ),
    )
})
}
