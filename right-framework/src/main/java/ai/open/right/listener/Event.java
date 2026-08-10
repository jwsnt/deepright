package ai.open.right.listener;

public interface Event extends EventDimension {

    public String getType();

    public Object getData();

    public Long getNow();

    public Event init();
}
