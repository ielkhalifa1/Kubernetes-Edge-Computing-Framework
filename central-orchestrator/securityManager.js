const crypto = require('crypto');

class SecurityManager {
  constructor(logger) {
    this.certificates = new Map();
    this.tokens = new Map();
    this.logger = logger;
  }

  // Generate a new token for node authentication
  generateToken(nodeId) {
    const token = crypto.randomBytes(32).toString('hex');
    const expiresAt = new Date();
    expiresAt.setDate(expiresAt.getDate() + 30); // 30 days expiration
    
    this.tokens.set(nodeId, {
      token,
      expiresAt,
      createdAt: new Date()
    });
    
    this.logger.info(`Generated new token for node: ${nodeId}`);
    return token;
  }

  // Validate token
  validateToken(nodeId, token) {
    const tokenInfo = this.tokens.get(nodeId);
    if (!tokenInfo) {
      return false;
    }
    
    if (tokenInfo.token !== token) {
      return false;
    }
    
    if (new Date() > tokenInfo.expiresAt) {
      return false;
    }
    
    return true;
  }

  // Revoke token
  revokeToken(nodeId) {
    if (this.tokens.has(nodeId)) {
      this.tokens.delete(nodeId);
      this.logger.info(`Revoked token for node: ${nodeId}`);
      return true;
    }
    return false;
  }
}

module.exports = { SecurityManager };