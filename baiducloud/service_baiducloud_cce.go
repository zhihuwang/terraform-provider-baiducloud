package baiducloud

import (
	"sort"

	"github.com/baidubce/bce-sdk-go/services/cce"
	"github.com/hashicorp/terraform/helper/resource"

	"github.com/terraform-providers/terraform-provider-baiducloud/baiducloud/connectivity"
)

type CceService struct {
	client *connectivity.BaiduClient
}

func (s *CceService) ClusterStateRefresh(clusterUuid string, failState []cce.ClusterStatus) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		action := "Query CCE Cluster " + clusterUuid
		raw, err := s.client.WithCCEClient(func(cceClient *cce.Client) (i interface{}, e error) {
			return cceClient.GetCluster(clusterUuid)
		})
		addDebug(action, raw)
		if err != nil {
			if NotFoundError(err) {
				return 0, string(cce.ClusterStatusDeleted), nil
			}
			return nil, "", WrapErrorf(err, DefaultErrorMsg, "baiducloud_cce_cluster", action, BCESDKGoERROR)
		}
		if raw == nil {
			return nil, "", nil
		}

		result := raw.(*cce.GetClusterResult)
		for _, statue := range failState {
			if result.Status == statue {
				return result, string(result.Status), WrapError(Error(GetFailTargetStatus, result.Status))
			}
		}

		addDebug(action, raw)
		return result, string(result.Status), nil
	}
}

func (s *CceService) GetCceV1Cluster(cluster_id string, client *connectivity.BaiduClient) (*cce.GetClusterResult, error) {
	action := "Get CCE v1 Cluster " + cluster_id

	raw, err := client.WithCCEClient(func(client *cce.Client) (i interface{}, e error) {
		return client.GetCluster(cluster_id)
	})

	if err != nil {
		return nil, WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_group_replica", action, BCESDKGoERROR)
	}
	response := raw.(*cce.GetClusterResult)
	return response, nil
}

func (s *CceService) GetCceV1Nodes(cluster_id string, client *connectivity.BaiduClient) (*cce.ListNodeResult, error) {
	action := "Get CCE Cluster v1 Nodes of" + cluster_id
	args := &cce.ListNodeArgs{ClusterUuid: cluster_id}
	raw, err := client.WithCCEClient(func(client *cce.Client) (i interface{}, e error) {
		return client.ListNodes(args)
	})
	if err != nil {
		return nil, WrapErrorf(err, DefaultErrorMsg, "baiducloud_ccev2_instance_group_replica", action, BCESDKGoERROR)
	}
	response := raw.(*cce.ListNodeResult)
	sort.Slice(response.Nodes[:], func(i, j int) bool {
		return response.Nodes[i].CreateTime.After(response.Nodes[j].CreateTime)
	})
	return response, nil
}
