package ai.open.right.workflow.config;

public interface ConfigService {

    public Config get(ConfigSearch configSearch) throws Exception;

    public Config search(ConfigSearch configSearch) throws Exception;
}
