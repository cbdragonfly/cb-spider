// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	_ "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVMHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vmHandler *GCPVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// Set VM Create Information
	// GCP 는 reqinfo에 ProjectID를 받아야 함.
	cblogger.Info(vmReqInfo)

	ctx := vmHandler.Ctx
	vmName := vmReqInfo.IId.NameId
	projectID := vmHandler.Credential.ProjectID
	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	//imageURL := "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024"
	imageURL := vmReqInfo.ImageIID.SystemId
	region := vmHandler.Region.Region

	zone := vmHandler.Region.Zone
	// email을 어디다가 넣지? 이것또한 문제넹
	clientEmail := vmHandler.Credential.ClientEmail

	//PublicIP처리
	// var publicIPAddress string
	// cblogger.Info("PublicIp 처리 시작")
	// publicIpHandler := GCPPublicIPHandler{
	// 	vmHandler.Region, vmHandler.Ctx, vmHandler.Client, vmHandler.Credential}

	//PublicIp를 전달 받았으면 전달 받은 Ip를 할당
	// if vmReqInfo.PublicIPId != "" {
	// 	cblogger.Info("PublicIp 정보 조회 시작")
	// 	publicIPInfo, err := publicIpHandler.GetPublicIP(vmReqInfo.PublicIPId)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.VMInfo{}, err
	// 	}
	// 	cblogger.Info("PublicIp 조회됨")
	// 	cblogger.Info(publicIPInfo)
	// 	publicIPAddress = publicIPInfo.PublicIP
	// } else { //PublicIp가 없으면 직접 생성
	// 	cblogger.Info("PublicIp 생성 시작")
	// 	// PublicIPHandler  불러서 처리 해야 함.
	// 	publicIpName := vmReqInfo.VMName
	// 	publicIpReqInfo := irs.PublicIPReqInfo{Name: publicIpName}
	// 	publicIPInfo, err := publicIpHandler.CreatePublicIP(publicIpReqInfo)

	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.VMInfo{}, err
	// 	}
	// 	cblogger.Info("PublicIp 생성됨")
	// 	cblogger.Info(publicIPInfo)
	// 	publicIPAddress = publicIPInfo.PublicIP
	// }

	//KEYPAIR HANDLER
	keypairHandler := GCPKeyPairHandler{
		vmHandler.Credential, vmHandler.Region}
	keypairInfo, errKeypair := keypairHandler.GetKey(vmReqInfo.KeyPairIID)
	pubKey := "cb-user:" + keypairInfo.PublicKey
	if errKeypair != nil {
		cblogger.Error(errKeypair)
		return irs.VMInfo{}, errKeypair
	}

	cblogger.Info("keypairInfo 정보")
	spew.Dump(keypairInfo)
	// Security Group Tags
	var securityTags []string

	for _, item := range vmReqInfo.SecurityGroupIIDs {
		securityTags = append(securityTags, item.NameId)
	}
	networkURL := prefix + "/global/networks/" + vmReqInfo.VpcIID.NameId
	subnetWorkURL := prefix + "/regions/" + region + "/subnetworks/" + vmReqInfo.SubnetIID.NameId
	instance := &compute.Instance{
		Name: vmName,
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{Key: "ssh-keys",
					Value: &pubKey},
			},
		},
		Labels: map[string]string{
			"keypair": strings.ToLower(vmReqInfo.KeyPairIID.NameId),
		},
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/" + vmReqInfo.VMSpecName,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    vmName + "-" + zone, //disk name 도 매번 바뀌어야 하는 값
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT", // default

					},
				},
				Network:    networkURL,
				Subnetwork: subnetWorkURL,
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: clientEmail,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
		Tags: &compute.Tags{
			Items: securityTags,
		},
	}

	cblogger.Info("VM 생성 시작")
	cblogger.Info(instance)
	spew.Dump(instance)
	op, err1 := vmHandler.Client.Instances.Insert(projectID, zone, instance).Do()
	cblogger.Info(op)
	spew.Dump(op)
	if err1 != nil {
		cblogger.Info("VM 생성 실패")
		cblogger.Error(err1)
		return irs.VMInfo{}, err1
	}

	/*
		js, err := op.MarshalJSON()
		if err != nil {
			cblogger.Info("VM 생성 실패")
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}

		cblogger.Info("Insert vm to marshal Json : ", string(js))
		cblogger.Infof("Got compute.Operation, err: %#v, %v", op, err)
	*/

	// 이게 시작하는  api Start 내부 매개변수로 projectID, zone, InstanceID
	//vm, err := vmHandler.Client.Instances.Start(project string, zone string, instance string)
	time.Sleep(time.Second * 10)
	vm, err2 := vmHandler.Client.Instances.Get(projectID, zone, vmName).Context(ctx).Do()
	if err2 != nil {
		cblogger.Error(err2)
		return irs.VMInfo{}, err2
	}
	//vmInfo := vmHandler.mappingServerInfo(vm)
	var securityTag []irs.IID

	for _, item := range vm.Tags.Items {
		iId := irs.IID{
			NameId:   item,
			SystemId: item,
		}

		securityTag = append(securityTag, iId)
	}
	var vpcHandler *GCPVPCHandler
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   vm.Name,
			SystemId: strconv.FormatUint(vm.Id, 10),
		},
		Region: irs.RegionInfo{
			Region: vmHandler.Region.Region,
			Zone:   vmHandler.Region.Zone,
		},
		VMUserId:          "cb-user",
		NetworkInterface:  vm.NetworkInterfaces[0].Name,
		SecurityGroupIIds: securityTag,
		VMSpecName:        vm.MachineType,
		KeyPairIId: irs.IID{
			NameId:   vm.Labels["keypair"],
			SystemId: vm.Labels["keypair"],
		},
		ImageIId:  vmHandler.getImageInfo(vm.Disks[0].Source),
		PublicIP:  vm.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		PrivateIP: vm.NetworkInterfaces[0].NetworkIP,
		VpcIID:    vmReqInfo.VpcIID,
		SubnetIID: vmReqInfo.SubnetIID,
		KeyValueList: []irs.KeyValue{
			{"SubNetwork", vm.NetworkInterfaces[0].Subnetwork},
			{"AccessConfigName", vm.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", vm.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"DiskDeviceName", vm.Disks[0].DeviceName},
			{"DiskName", vm.Disks[0].Source},
		},
	}

	return vmInfo, nil
}

// stop이라고 보면 될듯
func (vmHandler *GCPVMHandler) SuspendVM(vmID irs.IID) (irs.VMStatus, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Stop(projectID, zone, vmID.NameId).Context(ctx).Do()
	spew.Dump(inst)
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}

	fmt.Println("instance stop status :", inst.Status)
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *GCPVMHandler) ResumeVM(vmID irs.IID) (irs.VMStatus, error) {

	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Start(projectID, zone, vmID.NameId).Context(ctx).Do()
	spew.Dump(inst)
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}

	fmt.Println("instance resume status :", inst.Status)
	return irs.VMStatus("Resuming"), nil
}

func (vmHandler *GCPVMHandler) RebootVM(vmID irs.IID) (irs.VMStatus, error) {

	_, err := vmHandler.SuspendVM(vmID)
	if err != nil {
		return irs.VMStatus("Failed"), err
	}

	_, err2 := vmHandler.ResumeVM(vmID)
	if err2 != nil {
		return irs.VMStatus("Failed"), err2
	}

	return irs.VMStatus("Rebooting"), nil
}

func (vmHandler *GCPVMHandler) TerminateVM(vmID irs.IID) (irs.VMStatus, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	inst, err := vmHandler.Client.Instances.Delete(projectID, zone, vmID.NameId).Context(ctx).Do()
	spew.Dump(inst)
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}

	fmt.Println("instance status :", inst.Status)

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *GCPVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Items {
		if s.Name != "" {
			vmId := s.Name
			status, _ := vmHandler.GetVMStatus(irs.IID{NameId: vmId, SystemId: vmId})
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   vmId,
					SystemId: strconv.FormatUint(s.Id, 10),
				},

				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "PROVISIONING") {
		resultStatus = "Creating"
		//resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계에도 Pending이 있기 때문에 Pending은 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "RUNNING") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "STOPPING") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Terminated") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "STAGING") {
		resultStatus = "Resuming"
	} else {
		//resultStatus = "Failed"
		cblogger.Errorf("vmStatus [%s]와 일치하는 맵핑 정보를 찾지 못 함.", vmStatus)
		return irs.VMStatus("Failed"), errors.New(vmStatus + "와 일치하는 CB VM 상태정보를 찾을 수 없습니다.")
	}
	cblogger.Infof("VM 상태 치환 : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *GCPVMHandler) GetVMStatus(vmID irs.IID) (irs.VMStatus, error) { // GCP의 ID는 uint64 이므로 GCP에서는 Name을 ID값으로 사용한다.
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	instanceView, err := vmHandler.Client.Instances.Get(projectID, zone, vmID.NameId).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}

	// Get powerState, provisioningState
	//vmStatus := instanceView.Status
	vmStatus, errStatus := ConvertVMStatusString(instanceView.Status)
	//return irs.VMStatus(vmStatus), err
	return vmStatus, errStatus
}

func (vmHandler *GCPVMHandler) ListVM() ([]*irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var vmList []*irs.VMInfo
	for _, server := range serverList.Items {
		vmInfo := vmHandler.mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *GCPVMHandler) GetVM(vmID irs.IID) (irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmID.NameId).Do()
	spew.Dump(vm)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
}

// func getVmStatus(vl *compute.Service) string {
// 	var powerState, provisioningState string

// 	for _, stat := range vl {
// 		statArr := strings.Split(*stat.Code, "/")

// 		if statArr[0] == "PowerState" {
// 			powerState = statArr[1]
// 		} else if statArr[0] == "ProvisioningState" {
// 			provisioningState = statArr[1]
// 		}
// 	}

// 	// Set VM Status Info
// 	var vmState string
// 	if powerState != "" && provisioningState != "" {
// 		vmState = powerState + "(" + provisioningState + ")"
// 	} else if powerState != "" && provisioningState == "" {
// 		vmState = powerState
// 	} else if powerState == "" && provisioningState != "" {
// 		vmState = provisioningState
// 	} else {
// 		vmState = "-"
// 	}
// 	return vmState
// }

// func (vmHandler *GCPVMHandler) mappingServerInfo(server *compute.Instance) irs.VMInfo {
// 	//var gcpHanler *GCPVMHandler
// 	// Get Default VM Info

// 	// vmInfo := irs.VMInfo{
// 	// 	IId: irs.IID{
// 	// 		NameId:   server.Name,
// 	// 		SystemId: strconv.FormatUint(server.Id, 10),
// 	// 	},

// 	// 	Region: irs.RegionInfo{
// 	// 		Region: vmHandler.Region.Region,
// 	// 		Zone:   vmHandler.Region.Zone,
// 	// 	},
// 	// 	VMUserId:          "cb-user",
// 	// 	NetworkInterface:  server.NetworkInterfaces[0].Name,
// 	// 	SecurityGroupIIds: []irs.IID{
// 	// 		//server.Tags.Items,
// 	// 	},
// 	// 	VMSpecId: server.MachineType,
// 	// 	KeyPairIId: irs.IID{
// 	// 		NameId:   server.Labels["keypair"],
// 	// 		SystemId: server.Labels["keypair"],
// 	// 	},
// 	// 	ImageIId:  vmHandler.getImageInfo(server.Disks[0].Source),
// 	// 	PublicIP:  server.NetworkInterfaces[0].AccessConfigs[0].NatIP,
// 	// 	PrivateIP: server.NetworkInterfaces[0].NetworkIP,
// 	// 	VpcIID: irs.IID{
// 	// 		NameId: server.NetworkInterfaces[0].Network,
// 	// 	},
// 	// 	VirtualNetworkId: server.NetworkInterfaces[0].Network,
// 	// 	// SubNetworkID:       server.NetworkInterfaces[0].Subnetwork,
// 	// 	KeyValueList: []irs.KeyValue{
// 	// 		{"SubNetwork", server.NetworkInterfaces[0].Subnetwork},
// 	// 		{"AccessConfigName", server.NetworkInterfaces[0].AccessConfigs[0].Name},
// 	// 		{"NetworkTier", server.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
// 	// 		{"DiskDeviceName", server.Disks[0].DeviceName},
// 	// 		{"DiskName", server.Disks[0].Source},
// 	// 	},
// 	// }

// 	return vmInfo
// }
func (vmHandler *GCPVMHandler) getImageInfo(diskname string) irs.IID {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	dArr := strings.Split(diskname, "/")
	var result string
	if dArr != nil {
		result = dArr[len(dArr)-1]
	}
	cblogger.Infof("result : [%s]", result)

	info, err := vmHandler.Client.Disks.Get(projectID, zone, result).Do()
	spew.Dump(info)
	if err != nil {
		cblogger.Error(err)
		return irs.IID{}
	}
	//iArr := strings.Split(info.SourceImage, "/")
	iId := irs.IID{
		NameId:   info.Name,
		SystemId: strconv.FormatUint(info.Id, 10),
	}
	return iId
}

func (vmHandler *GCPVMHandler) getKeyPairInfo(diskname string) irs.IID {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	dArr := strings.Split(diskname, "/")
	var result string
	if dArr != nil {
		result = dArr[len(dArr)-1]
	}
	cblogger.Infof("result : [%s]", result)

	info, err := vmHandler.Client.Disks.Get(projectID, zone, result).Do()
	spew.Dump(info)
	if err != nil {
		cblogger.Error(err)
		return ""
	}
	iArr := strings.Split(info.SourceImage, "/")
	return iArr[len(iArr)-1]
}
