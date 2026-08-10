package ai.open.right.workflow.flow.file.impl;

import ai.open.right.workflow.flow.file.FileStore;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.ArrayUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

@Getter
@Setter
abstract public class SafeStore implements FileStore {

    protected Integer oversize = 20971520;

    public void check(byte[] bytes) throws Exception {
        int len = ArrayUtils.getLength(bytes);
        Assert.isTrue(len <= this.oversize, "The file is oversized: " + len + "/" + this.oversize + ", please config `file.store.oversize`");
    }

    @Getter
    @Setter
    public static class DefInitConfig {

        @Value("${file.store.oversize:20971520}")
        protected Integer oversize;
    }
}
