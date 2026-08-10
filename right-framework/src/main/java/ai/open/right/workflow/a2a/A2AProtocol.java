package ai.open.right.workflow.a2a;

public interface A2AProtocol {

    public static final String METHOD = "internal";

    public Boolean isSupport(String internal);

    // 清除标记
    public A2AProtocol reset();
}
