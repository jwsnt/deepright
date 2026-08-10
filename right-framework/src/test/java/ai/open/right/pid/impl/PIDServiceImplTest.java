package ai.open.right.pid.impl;

import ai.open.right.pid.PIDService;
import org.apache.commons.io.IOUtils;
import org.junit.Assert;
import org.junit.Test;

import java.io.File;
import java.io.FileInputStream;

public class PIDServiceImplTest {

    @Test
    public void testPid() throws Exception {
        PIDServiceImpl pidService = new PIDServiceImpl();
        pidService.init();
        Assert.assertNotNull(pidService.getPid());
        Assert.assertNotNull(pidService.pid());
    }

    @Test
    public void testFile() throws Exception {
        PIDServiceImpl pidService = new PIDServiceImpl();
        pidService.setFile("src/test/resources/pid");
        pidService.init();
        File file = new File("src/test/resources/pid");
        Assert.assertNotNull(IOUtils.toString(new FileInputStream(file), "UTF-8"));
        file.deleteOnExit();
    }

    @Test
    public void testInit() throws Exception {
        PIDServiceImpl.InitConfig initConfig = new PIDServiceImpl.InitConfig();
        initConfig.setFile("PATH");
        PIDServiceImpl pidService = (PIDServiceImpl)initConfig.pidService();
        Assert.assertEquals(pidService.getFile(), initConfig.getFile());
    }

    @org.junit.jupiter.api.Test
    public void testInitNoFile() throws Exception {
        PIDServiceImpl pidService = new PIDServiceImpl();
        pidService.setFile(null);
        pidService.init();
        org.junit.jupiter.api.Assertions.assertNotNull(pidService.pid());
    }
}
