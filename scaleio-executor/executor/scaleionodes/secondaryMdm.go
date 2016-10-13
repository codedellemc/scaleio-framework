package core

import (
	"time"

	log "github.com/Sirupsen/logrus"
	xplatform "github.com/dvonthenen/goxplatform"

	basenode "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/basenode"
	common "github.com/codedellemc/scaleio-framework/scaleio-executor/executor/common"
	types "github.com/codedellemc/scaleio-framework/scaleio-scheduler/types"
)

//ScaleioSecondaryMdmNode implementation for ScaleIO Secondary MDM Node
type ScaleioSecondaryMdmNode struct {
	basenode.MdmScaleioNode
}

//NewSec generates a Secondary MDM Node object
func NewSec() *ScaleioSecondaryMdmNode {
	myNode := &ScaleioSecondaryMdmNode{}
	return myNode
}

//RunStateUnknown default action for StateUnknown
func (ssmn *ScaleioSecondaryMdmNode) RunStateUnknown(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	reboot, err := EnvironmentSetup(state)
	if err != nil {
		log.Errorln("EnvironmentSetup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := UpdateNodeState(types.StateCleanPrereqsReboot)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateCleanPrereqsReboot")
	}

	state = common.WaitForCleanPrereqsReboot(spmn.UpdateScaleIOState())

	errState = UpdateNodeState(types.StatePrerequisitesInstalled)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StatePrerequisitesInstalled")
	}

	//requires a reboot?
	if reboot {
		log.Infoln("Reboot required before StatePrerequisitesInstalled!")

		time.Sleep(time.Duration(DelayForRebootInSeconds) * time.Second)

		rebootErr := xplatform.GetInstance().Run.Command(rebootCmdline, rebootCheck, "")
		if rebootErr != nil {
			log.Errorln("Install Kernel Failed:", rebootErr)
		}

		time.Sleep(time.Duration(WaitForRebootInSeconds) * time.Second)
	} else {
		log.Infoln("No need to reboot while installing prerequisites")
	}
}

//RunStatePrerequisitesInstalled default action for StatePrerequisitesInstalled
func (ssmn *ScaleioSecondaryMdmNode) RunStatePrerequisitesInstalled(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	state = common.WaitForPrereqsFinish(spmn.UpdateScaleIOState())
	err := ManagementSetup(state, true)
	if err != nil {
		log.Errorln("ManagementSetup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	err = NodeSetup(state)
	if err != nil {
		log.Errorln("NodeSetup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := UpdateNodeState(types.StateBasePackagedInstalled)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateBasePackagedInstalled")
	}
}

//RunStateBasePackagedInstalled default action for StateBasePackagedInstalled
func (ssmn *ScaleioSecondaryMdmNode) RunStateBasePackagedInstalled(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	state = common.WaitForBaseFinish(spmn.UpdateScaleIOState())

	errState := UpdateNodeState(types.StateInitializeCluster)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateInitializeCluster")
	}
}

//RunStateInitializeCluster default action for StateInitializeCluster
func (ssmn *ScaleioSecondaryMdmNode) RunStateInitializeCluster(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	state = common.WaitForClusterInstallFinish(spmn.UpdateScaleIOState())
	reboot, err := GatewaySetup(state)
	if err != nil {
		log.Errorln("GatewaySetup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}
	ssmn.RebootRequired = reboot

	errState := UpdateNodeState(types.StateInstallRexRay)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateInstallRexRay")
	}
}

//RunStateInstallRexRay default action for StateInstallRexRay
func (ssmn *ScaleioSecondaryMdmNode) RunStateInstallRexRay(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	state = common.WaitForClusterInitializeFinish(spmn.UpdateScaleIOState())
	reboot, err := RexraySetup(state)
	if err != nil {
		log.Errorln("REX-Ray setup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	err = SetupIsolator(state)
	if err != nil {
		log.Errorln("Mesos Isolator setup Failed:", err)
		errState := UpdateNodeState(types.StateFatalInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFatalInstall")
		}
		return
	}

	errState := UpdateNodeState(types.StateCleanInstallReboot)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateCleanInstallReboot")
	}

	state = common.WaitForCleanInstallReboot(spmn.UpdateScaleIOState())

	//requires a reboot?
	if ssmn.RebootRequired || reboot {
		log.Infoln("Reboot required before StateFinishInstall!")
		log.Debugln("rebootRequired:", rebootRequired)
		log.Debugln("reboot:", reboot)

		time.Sleep(time.Duration(DelayForRebootInSeconds) * time.Second)

		errState = UpdateNodeState(types.StateSystemReboot)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateSystemReboot")
		}

		rebootErr := xplatform.GetInstance().Run.Command(rebootCmdline, rebootCheck, "")
		if rebootErr != nil {
			log.Errorln("Install Kernel Failed:", rebootErr)
		}

		time.Sleep(time.Duration(WaitForRebootInSeconds) * time.Second)
	} else {
		log.Infoln("No need to reboot while installing REX-Ray")

		errState = UpdateNodeState(types.StateFinishInstall)
		if errState != nil {
			log.Errorln("Failed to signal state change:", errState)
		} else {
			log.Debugln("Signaled StateFinishInstall")
		}
	}
}

//RunStateSystemReboot default action for StateSystemReboot
func (ssmn *ScaleioSecondaryMdmNode) RunStateSystemReboot(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	errState := UpdateNodeState(types.StateFinishInstall)
	if errState != nil {
		log.Errorln("Failed to signal state change:", errState)
	} else {
		log.Debugln("Signaled StateFinishInstall")
	}
}

//RunStateFinishInstall default action for StateFinishInstall
func (ssmn *ScaleioSecondaryMdmNode) RunStateFinishInstall(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	log.Debugln("In StateFinishInstall. Wait for", PollForChangesInSeconds,
		"seconds for changes in the cluster.")
	time.Sleep(time.Duration(PollForChangesInSeconds) * time.Second)

	if state.DemoMode {
		log.Infoln("DemoMode = TRUE. Leaving marker file for previously configured")
		LeaveMarkerFileForConfigured(node)
	}

	//TODO eventual plan for MDM node behavior
	/*
		if clusterStatusBad then
			doClusterRemediate()
		else if upgrade then
			_ = common.WaitForClusterUpgrade(spmn.UpdateScaleIOState())
			doUpgrade()
		else
			checkForNewDataNodesToAdd()
	*/

	//TODO replace this at some point with API calls instead of CLI
	pri, errPri := common.GetPrimaryMdmNode(state)

	if errPri != nil {
		log.Errorln("Unable to find the Primary MDM Node. Retry again later.", errPri)
	} else {
		if (pri.LastContact + OfflineTimeForMdmNodesInSeconds) < time.Now().Unix() {
			//This is the checkForNewDataNodesToAdd(). Other functionality TBD.
			err := AddSdsNodesToCluster(state, true)
			if err != nil {
				log.Errorln("Failed to add node to ScaleIO cluster:", err)
			}
		}
	}
}

//RunStateUpgradeCluster default action for StateUpgradeCluster
func (ssmn *ScaleioSecondaryMdmNode) RunStateUpgradeCluster(state *types.ScaleIOFramework, node *types.ScaleIONode) {
	log.Debugln("In StateUpgradeCluster. Do nothing.")
	//TODO process the upgrade here
}
