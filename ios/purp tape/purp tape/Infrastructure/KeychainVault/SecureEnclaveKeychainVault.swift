import Foundation
import Security

public enum KeychainVaultError: Error, Sendable {
    case encodingFailed
    case decodingFailed
    case encryptionFailed
    case decryptionFailed
    case keyGenerationFailed
    case keychainError(OSStatus)
}

public actor SecureEnclaveKeychainVault: KeychainVault {
    private let service: String
    private let account = "auth_session_v1"
    private let keyTag: Data
    
    public init(service: String = "ilyes.purp-tape.auth") {
        self.service = service
        self.keyTag = Data("\(service).secure-enclave.key".utf8)
    }
    
    public func saveSession(_ session: AuthSession) async throws {
        let payload = try JSONEncoder().encode(session)
        let encrypted = try encrypt(payload)
        
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        
        let attributes: [String: Any] = [
            kSecValueData as String: encrypted,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        
        let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if updateStatus == errSecSuccess { return }
        
        if updateStatus == errSecItemNotFound {
            var addQuery = query
            attributes.forEach { addQuery[$0.key] = $0.value }
            let addStatus = SecItemAdd(addQuery as CFDictionary, nil)
            guard addStatus == errSecSuccess else {
                throw KeychainVaultError.keychainError(addStatus)
            }
            return
        }
        
        throw KeychainVaultError.keychainError(updateStatus)
    }
    
    public func loadSession() async throws -> AuthSession? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        
        var item: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        
        if status == errSecItemNotFound {
            return nil
        }
        
        guard status == errSecSuccess else {
            throw KeychainVaultError.keychainError(status)
        }
        
        guard let data = item as? Data else {
            throw KeychainVaultError.decodingFailed
        }
        
        let decrypted = try decrypt(data)
        do {
            return try JSONDecoder().decode(AuthSession.self, from: decrypted)
        } catch {
            throw KeychainVaultError.decodingFailed
        }
    }
    
    public func clearSession() async throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        
        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainVaultError.keychainError(status)
        }
    }
    
    private func encrypt(_ plaintext: Data) throws -> Data {
        let privateKey = try loadOrCreatePrivateKey()
        
        guard let publicKey = SecKeyCopyPublicKey(privateKey) else {
            throw KeychainVaultError.encryptionFailed
        }
        
        let algorithm: SecKeyAlgorithm = .eciesEncryptionCofactorX963SHA256AESGCM
        guard SecKeyIsAlgorithmSupported(publicKey, .encrypt, algorithm) else {
            return plaintext
        }
        
        var error: Unmanaged<CFError>?
        guard let encrypted = SecKeyCreateEncryptedData(publicKey, algorithm, plaintext as CFData, &error) as Data? else {
            if error != nil {
                throw KeychainVaultError.encryptionFailed
            }
            return plaintext
        }
        
        return encrypted
    }
    
    private func decrypt(_ ciphertext: Data) throws -> Data {
        let privateKey = try loadOrCreatePrivateKey()
        
        let algorithm: SecKeyAlgorithm = .eciesEncryptionCofactorX963SHA256AESGCM
        guard SecKeyIsAlgorithmSupported(privateKey, .decrypt, algorithm) else {
            return ciphertext
        }
        
        var error: Unmanaged<CFError>?
        if let decrypted = SecKeyCreateDecryptedData(privateKey, algorithm, ciphertext as CFData, &error) as Data? {
            return decrypted
        }
        
        if error == nil {
            return ciphertext
        }
        
        throw KeychainVaultError.decryptionFailed
    }
    
    private func loadOrCreatePrivateKey() throws -> SecKey {
        let query: [String: Any] = [
            kSecClass as String: kSecClassKey,
            kSecAttrApplicationTag as String: keyTag,
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecReturnRef as String: true,
        ]
        
        var item: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        if status == errSecSuccess {
            guard let item else {
                throw KeychainVaultError.keyGenerationFailed
            }
            guard CFGetTypeID(item) == SecKeyGetTypeID() else {
                throw KeychainVaultError.keyGenerationFailed
            }
            return unsafeDowncast(item, to: SecKey.self)
        }
        
        if status != errSecItemNotFound {
            throw KeychainVaultError.keychainError(status)
        }
        
        var accessError: Unmanaged<CFError>?
        guard let access = SecAccessControlCreateWithFlags(
            kCFAllocatorDefault,
            kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            [.privateKeyUsage],
            &accessError
        ) else {
            throw KeychainVaultError.keyGenerationFailed
        }
        
        let attributes: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecAttrTokenID as String: kSecAttrTokenIDSecureEnclave,
            kSecPrivateKeyAttrs as String: [
                kSecAttrIsPermanent as String: true,
                kSecAttrApplicationTag as String: keyTag,
                kSecAttrAccessControl as String: access,
            ],
        ]
        
        var generationError: Unmanaged<CFError>?
        if let key = SecKeyCreateRandomKey(attributes as CFDictionary, &generationError) {
            return key
        }
        
        let fallbackAttributes: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecPrivateKeyAttrs as String: [
                kSecAttrIsPermanent as String: true,
                kSecAttrApplicationTag as String: keyTag,
                kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            ],
        ]
        
        var fallbackError: Unmanaged<CFError>?
        guard let fallbackKey = SecKeyCreateRandomKey(fallbackAttributes as CFDictionary, &fallbackError) else {
            throw KeychainVaultError.keyGenerationFailed
        }
        
        return fallbackKey
    }
}
