package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/redis/go-redis/v9"
)

type ICacheService interface {
	GetClusterInfo(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, bool)
	SetClusterInfo(ctx context.Context, clusterID string, info *dto.ClusterInfoResponse, ttl time.Duration) error
	GetClusterStats(ctx context.Context, clusterID string) (*dto.ClusterStatsResponse, bool)
	SetClusterStats(ctx context.Context, clusterID string, stats *dto.ClusterStatsResponse, ttl time.Duration) error
	GetReplicationStatus(ctx context.Context, clusterID string) (*dto.ReplicationStatusResponse, bool)
	SetReplicationStatus(ctx context.Context, clusterID string, status *dto.ReplicationStatusResponse, ttl time.Duration) error
	InvalidateCluster(ctx context.Context, clusterID string) error
	InvalidateClusterInfo(ctx context.Context, clusterID string) error
}

type cacheService struct {
	redis *redis.Client
}

func NewCacheService(redis *redis.Client) ICacheService {
	return &cacheService{redis: redis}
}

func (s *cacheService) GetClusterInfo(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, bool) {
	key := fmt.Sprintf("cluster:info:%s", clusterID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var info dto.ClusterInfoResponse
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, false
	}

	return &info, true
}

func (s *cacheService) SetClusterInfo(ctx context.Context, clusterID string, info *dto.ClusterInfoResponse, ttl time.Duration) error {
	key := fmt.Sprintf("cluster:info:%s", clusterID)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, ttl).Err()
}

func (s *cacheService) GetClusterStats(ctx context.Context, clusterID string) (*dto.ClusterStatsResponse, bool) {
	key := fmt.Sprintf("cluster:stats:%s", clusterID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var stats dto.ClusterStatsResponse
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, false
	}

	return &stats, true
}

func (s *cacheService) SetClusterStats(ctx context.Context, clusterID string, stats *dto.ClusterStatsResponse, ttl time.Duration) error {
	key := fmt.Sprintf("cluster:stats:%s", clusterID)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, ttl).Err()
}

func (s *cacheService) GetReplicationStatus(ctx context.Context, clusterID string) (*dto.ReplicationStatusResponse, bool) {
	key := fmt.Sprintf("cluster:replication:%s", clusterID)
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var status dto.ReplicationStatusResponse
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, false
	}

	return &status, true
}

func (s *cacheService) SetReplicationStatus(ctx context.Context, clusterID string, status *dto.ReplicationStatusResponse, ttl time.Duration) error {
	key := fmt.Sprintf("cluster:replication:%s", clusterID)
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, ttl).Err()
}

func (s *cacheService) InvalidateCluster(ctx context.Context, clusterID string) error {
	keys := []string{
		fmt.Sprintf("cluster:info:%s", clusterID),
		fmt.Sprintf("cluster:stats:%s", clusterID),
		fmt.Sprintf("cluster:replication:%s", clusterID),
	}

	return s.redis.Del(ctx, keys...).Err()
}

func (s *cacheService) InvalidateClusterInfo(ctx context.Context, clusterID string) error {
	key := fmt.Sprintf("cluster:info:%s", clusterID)
	return s.redis.Del(ctx, key).Err()
}
