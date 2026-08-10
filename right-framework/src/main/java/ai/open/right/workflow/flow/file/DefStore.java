package ai.open.right.workflow.flow.file;

import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.impl.SysStore;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.Map;

@Slf4j
@Getter
@Setter
public class DefStore implements FileStore {

    public static final String NAME = "file.store.def";

    private Map<String, FileStore> fileStore;

    private String def = SysStore.NAME;

    public String store(byte[] bytes, String suffix, WorkflowTask workTask, String name) throws Exception {
        return this.fetchStore(name).store(bytes, suffix, workTask);
    }

    @Override
    public String store(byte[] bytes, String suffix, WorkflowTask workTask) throws Exception {
        return this.fetchStore().store(bytes, suffix, workTask);
    }

    public String store(byte[] bytes, String suffix, String name) throws Exception {
        return this.fetchStore(name).store(bytes, suffix);
    }

    @Override
    public String store(byte[] bytes, String suffix) throws Exception {
        return this.fetchStore().store(bytes, suffix);
    }

    public Boolean supportFunction(String name) throws Exception {
        return MapUtils.getObject(this.fileStore, name) != null;
    }

    @Override
    public Boolean supportNetwork() throws Exception {
        return this.fetchStore().supportNetwork();
    }

    @Override
    public Boolean supportFilesys() throws Exception {
        return this.fetchStore().supportFilesys();
    }

    @Override
    public String name() throws Exception {
        Assert.hasText(this.def, "The file store (def) can not be empty");
        return this.def;
    }

    public FileStore fetchStore(String name) throws Exception {
        FileStore fileStore = this.fileStore.get(name);
        Assert.notNull(fileStore, "The file store can not be empty: " + name);
        return fileStore;
    }

    public FileStore fetchStore() throws Exception {
        return this.fetchStore(this.def);
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        private Map<String, FileStore> fileStore;

        @Value("${file.store.def:file.store.sys}")
        private String def = SysStore.NAME;

        @Bean(DefStore.NAME)
        @ConditionalOnMissingBean(name = DefStore.NAME)
        public DefStore defStore() throws Exception {
            DefStore defStore = new DefStore();
            BeanUtils.copyProperties(this, defStore);
            log.info("DefStore inited");
            return defStore;
        }
    }
}
