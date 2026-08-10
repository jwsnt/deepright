package ai.open.right.release;

import ai.open.right.ObjectBuilder;
import org.apache.commons.io.FileUtils;
import org.easymock.EasyMock;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import org.springframework.boot.system.ApplicationHome;

import java.io.File;

public class ResourceReleaserTest {

    @Test
    public void testJar() throws Exception {
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            public String getUrl() throws Exception {
                return "jar:nested:/Your Path/target/xxx.jar/!BOOT-INF/classes/!/";
            }
        };
        Assertions.assertEquals("/Your Path/target/xxx.jar", resourceConfig.getJar().getAbsolutePath());
    }

    @Test
    public void testApplication() throws Exception {
        ResourceReleaser resourceConfig = new ResourceReleaser();
        resourceConfig.setResourceService(ObjectBuilder.buildResourceService(ResourceReleaser.class));
        resourceConfig.init();
        Assertions.assertNotNull(resourceConfig.getHome());
    }

    @Test
    public void testUrl() throws Exception {
        ApplicationHome applicationHome = EasyMock.createMock(ApplicationHome.class);
        EasyMock.expect(applicationHome.getSource()).andReturn(new File(".")).anyTimes();
        EasyMock.replay(applicationHome);
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            protected ApplicationHome getHome() throws Exception {
                return applicationHome;
            }
        };
        resourceConfig.setResourceService(ObjectBuilder.buildResourceService(ResourceReleaser.class));
        resourceConfig.init();
        Assertions.assertNotNull(resourceConfig.getResourceService());
        Assertions.assertTrue(resourceConfig.getJar().toString().contains("right-framework"));
        EasyMock.verify(applicationHome);
    }

    @Test
    public void testInit() throws Exception {
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            public File getJar() throws Exception {
                return new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0.jar");
            }
        };
        resourceConfig.setRelease(true);
        resourceConfig.init();
        FileUtils.deleteDirectory(new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0"));
    }

    @Test
    public void testFailed() throws Exception {
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            public File getJar() throws Exception {
                return new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0.zip");
            }
        };
        resourceConfig.setRelease(true);
        resourceConfig.init();
    }

    @Test
    public void testResourceConfig() {
        ResourceReleaser config = new ResourceReleaser();
        Assertions.assertNotNull(config);
    }

    @Test
    public void testInitException() {
        // 覆盖 init() 中的 catch (Exception e) 分支
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            protected File getJar() throws Exception {
                throw new Exception("Mock Exception for Coverage");
            }
        };
        resourceConfig.setRelease(true);
        // 验证不会抛出异常（被内部 catch 捕获并记录日志）
        Assertions.assertDoesNotThrow(resourceConfig::init);
    }

    @Test
    public void testGetJarNull() throws Exception {
        // 覆盖 getJar() 中的 return null 分支
        
        // 分支 1: url 不以 "jar:" 开头
        ResourceReleaser config1 = new ResourceReleaser() {
            @Override
            protected String getUrl() throws Exception {
                return "file:/path/to/resource";
            }
        };
        Assertions.assertNull(config1.getJar());

        // 分支 2: url 以 "jar:" 开头但没有 "!"
        ResourceReleaser config2 = new ResourceReleaser() {
            @Override
            protected String getUrl() throws Exception {
                return "jar:file:/path/to/resource";
            }
        };
        Assertions.assertNull(config2.getJar());
    }
    
    @Test
    public void testInitNoRelease() {
        // 覆盖 release 为 false 的分支
        ResourceReleaser resourceConfig = new ResourceReleaser();
        resourceConfig.setRelease(false);
        resourceConfig.init();
        Assertions.assertFalse(resourceConfig.getRelease());
    }

    @Test
    public void testInitWithSuffix() throws Exception {
        ResourceReleaser resourceConfig = new ResourceReleaser() {
            @Override
            public File getJar() throws Exception {
                return new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0.jar");
            }
        };
        resourceConfig.setRelease(true);
        resourceConfig.setSuffix(".class,.xml");
        resourceConfig.init();
        FileUtils.deleteDirectory(new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0"));
    }
    
    @Test
    public void testInitWithFile() throws Exception {
        String testJarPath = System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0.jar";
        ResourceReleaser resourceConfig = new ResourceReleaser();
        resourceConfig.setRelease(true);
        resourceConfig.setFile(testJarPath);
        resourceConfig.init();
        FileUtils.deleteDirectory(new File(System.getProperty("user.dir") + "/src/test/resources/right-demo-1.0"));
    }
}
