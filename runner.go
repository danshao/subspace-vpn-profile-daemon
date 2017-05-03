package main

import (
	"fmt"
	"time"
	"gitlab.ecoworkinc.com/subspace/softetherlib/softether"

	"gitlab.ecoworkinc.com/subspace/subspace-utility/subspace/model"
	"gitlab.ecoworkinc.com/subspace/subspace-utility/subspace/repository"
	"gitlab.ecoworkinc.com/subspace/subspace-utility/subspace/utils"
)

type ProfileDaemonRunner struct {
	Server                    softether.SoftEther
	ProfileSnapshotRepository repository.MysqlProfileSnapshotRepository
	ProfileRepository         repository.MysqlProfileRepository
}

const INTERVAL = 2 * time.Second
const REDIS_TIME_TO_LIVE = 30 * time.Second

var ticker *time.Ticker

func (daemon ProfileDaemonRunner) Start() {
	ticker = time.NewTicker(INTERVAL)

	go func() {
		for t := range ticker.C {

			fmt.Println("Start fetch subspace status at", time.Now())
			userList, code := daemon.Server.GetUserList()

			if 0 != code {
				fmt.Println(t, "Get UserList fail code:", code)
				// Pending if SoftEther is too busy
				daemon.pending()
				continue
			}

			profileBriefs := make([]*model.SoftEtherUserBrief, 0)
			for _, rawData := range userList {
				if profileBrief, err := utils.ParseUserList(daemon.Server.Hub, rawData); nil == err {
					profileBriefs = append(profileBriefs, profileBrief)
				} else {
					fmt.Println(t, "UserList format is not expected")
				}
			}

			fmt.Println("User List Start")
			profileSnapshots := make([]*model.ProfileSnapshot, 0)
			for _, brief := range profileBriefs {
				fmt.Println("User Get ", brief.UserName)

				if userDetail, code := daemon.Server.GetUserInfo(brief.UserName); 0 == code {
					if profile, err := utils.ParseUserGet(daemon.Server.Hub, userDetail); nil == err {
						//由於 UserGet 沒有 Last Login 資料，因此需要整合 UserList 拿到的資料
						profile.MergeSoftEtherUserBrief(brief)
						profileSnapshots = append(profileSnapshots, profile)
					} else {
						fmt.Println(t, "UserGet format is not expected")
					}
				} else {
					fmt.Println(t, "Get UserGet fail code:", code)
				}
			}

			if err := daemon.ProfileSnapshotRepository.InsertBatch(profileSnapshots); nil != err {
				fmt.Println(err)
			}
			if err := daemon.ProfileRepository.UpdateBatch(profileSnapshots); nil != err {
				fmt.Println(err)
			}
		}
	}()
}

func (daemon ProfileDaemonRunner) Stop() {
	ticker.Stop()
}

func (daemon ProfileDaemonRunner) pending() {
	time.Sleep(REDIS_TIME_TO_LIVE / 5)
}
