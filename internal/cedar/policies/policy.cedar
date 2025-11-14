// Cedar Policies for Document Management System

// ポリシー0: 地理的制限 - 日本国内IPまたはプライベートIPのみアクセス許可
forbid(
    principal,
    action,
    resource
)
unless {
    context.is_japan_ip || context.is_private_ip
};

// ポリシー1: 管理者は全ての操作が可能
permit(
    principal,
    action,
    resource
)
when {
    principal.role == "admin"
};

// ポリシー2: エディターはドキュメントの一覧表示、取得、作成、更新が可能
permit(
    principal,
    action in [
        DocumentApp::Action::"ListDocuments",
        DocumentApp::Action::"GetDocument",
        DocumentApp::Action::"CreateDocument",
        DocumentApp::Action::"UpdateDocument"
    ],
    resource
)
when {
    principal.role == "editor"
};

// ポリシー3: 閲覧者はドキュメントの一覧表示と取得のみ可能
permit(
    principal,
    action in [
        DocumentApp::Action::"ListDocuments",
        DocumentApp::Action::"GetDocument"
    ],
    resource
)
when {
    principal.role == "viewer"
};

// ポリシー4: ドキュメントのオーナーは自分のドキュメントを削除可能
permit(
    principal,
    action == DocumentApp::Action::"DeleteDocument",
    resource
)
when {
    resource.owner == principal
};
