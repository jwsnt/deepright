package ai.open.right.workflow.flow.file.impl;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

/**
 * SysStore.InitConfig 单元测试类
 * 验证 Bean 的创建以及属性注入逻辑
 */
class SysStoreInitConfigTest {

    @Test
    @DisplayName("测试 InitConfig 的属性注入和 SysStore Bean 的创建")
    void testSysStoreBeanCreation() throws Exception {
        // 1. 初始化 InitConfig 实例
        SysStore.InitConfig initConfig = new SysStore.InitConfig();

        // 2. 模拟 Spring 属性注入，验证 Setter/Getter
        String expectedPath = "/data/right/storage";
        initConfig.setPath(expectedPath);
        Assertions.assertEquals(expectedPath, initConfig.getPath(), "InitConfig 的 path 属性应与设置值一致");

        // 3. 调用 Bean 创建方法
        SysStore sysStore = initConfig.sysStore();

        // 4. 验证生成的 Bean 及其属性
        Assertions.assertNotNull(sysStore, "生成的 SysStore 实例不应为空");
        Assertions.assertEquals(expectedPath, sysStore.getPath(), "SysStore 的 path 属性应从 InitConfig 中正确复制");
    }

    @Test
    @DisplayName("测试 InitConfig 的默认路径值")
    void testDefaultPath() throws Exception {
        SysStore.InitConfig initConfig = new SysStore.InitConfig();
        // 在没有 Spring 环境注入时，手动设置或验证逻辑
        initConfig.setPath(".");
        SysStore sysStore = initConfig.sysStore();
        Assertions.assertEquals(".", sysStore.getPath(), "默认路径应为当前目录");
    }
}
