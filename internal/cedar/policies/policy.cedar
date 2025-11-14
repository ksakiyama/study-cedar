// Cedar Policies for Document Management System

// Policy 0: Geographic restriction - Allow access only from Japan IPs or private IPs
forbid(
    principal,
    action,
    resource
)
unless {
    context.is_japan_ip || context.is_private_ip
};

// Policy 1: Admins can perform all operations (bypasses group restrictions)
permit(
    principal,
    action,
    resource
)
when {
    principal.role == "admin"
};

// Policy 2: Editors with group access can list, view, create, and update documents
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
    principal.role == "editor" &&
    context.has_group_access
};

// Policy 3: Viewers with group access can only list and view documents
permit(
    principal,
    action in [
        DocumentApp::Action::"ListDocuments",
        DocumentApp::Action::"GetDocument"
    ],
    resource
)
when {
    principal.role == "viewer" &&
    context.has_group_access
};

// Policy 4: Document owners can delete their own documents
permit(
    principal,
    action == DocumentApp::Action::"DeleteDocument",
    resource
)
when {
    resource.owner == principal
};
