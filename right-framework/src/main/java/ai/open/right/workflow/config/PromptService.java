package ai.open.right.workflow.config;

public interface PromptService {

    public Prompt get(PromptSearch promptSearch) throws Exception;

    public Prompt search(PromptSearch promptSearch) throws Exception;
}
