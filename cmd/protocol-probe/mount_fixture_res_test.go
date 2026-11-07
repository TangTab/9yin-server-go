package main

import "testing"

// 坐骑目录的资源供给回归钉（jingmai_fixture_res_test 同款根因防护）：
// tool_item.ini / itemartstatic.ini / new_ride.ini 必须能从三层资源仓
// 解析——2026-11-07 补齐前 tool_item/new_ride 不在夹具仓，CurRiderType
// 整域静默降级为 0。role 618 官服迁移档的两只坐骑（41523/9513）是
// 真表锚点。
func TestMountCatalogResolvesFromResourceStore(t *testing.T) {
	catalog := loadMountCatalog(defaultToolItemINIPath, defaultItemArtStaticPath, defaultNewRideINIPath)
	if len(catalog.byPack) < 50 {
		t.Fatalf("ridable mounts=%d, want >=50 (real table)", len(catalog.byPack))
	}
	if got := catalog.riderTypeOf(41523); got == 0 {
		t.Fatalf("riderType(41523 MOUNT_HAWK_003)=0, want non-zero")
	}
	if got := catalog.riderTypeOf(9513); got == 0 {
		t.Fatalf("riderType(9513 HAIWAI_MOUNT_6_009)=0, want non-zero")
	}
}
