import SwiftUI

struct SocialAuthButton: View {
    let icon: String
    let label: String
    let action: () -> Void
    
    var body: some View {
        PurpTapeSecondaryButton(label, icon: icon, action: action)
    }
}
