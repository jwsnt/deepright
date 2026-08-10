package ai.open.right.workflow.notify;

public interface NotifierTrack {

    public void beginFunCallTrack(String funCallTrack);

    public void beginFunCallTrack();

    public void beginChatTrack();

    public void closeFunCallTrack();

    public String getFunCallTrack();

    public Boolean containFunCallTrack();

    public Boolean containChatTrack();
}
